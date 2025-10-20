package openapi2mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/kong/go-apiops/logbasics"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	openapibase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// O2MCPOptions defines the options for an OpenAPI to MCP conversion operation
type O2MCPOptions struct {
	// Route name/ID for the ai-mcp-proxy plugin (mutually exclusive with ServiceName)
	RouteName string
	// Service name/ID for the ai-mcp-proxy plugin (mutually exclusive with RouteName)
	ServiceName string
	// Mode for the MCP proxy
	Mode string
	// Server timeout in milliseconds
	ServerTimeout int
	// Whether to log statistics
	LogStatistics bool
	// Whether to log payloads
	LogPayloads bool
	// Whether to forward client headers
	ForwardClientHeaders bool
	// Ignore circular references
	IgnoreCircularRefs bool
}

// setDefaults sets the defaults for the OpenAPI2MCP operation
func (opts *O2MCPOptions) setDefaults() {
	if opts.Mode == "" {
		opts.Mode = "conversion-listener"
	}
	if opts.ServerTimeout == 0 {
		opts.ServerTimeout = 60000
	}
	// ForwardClientHeaders defaults to true if not explicitly set
	if !opts.LogStatistics && !opts.LogPayloads {
		opts.ForwardClientHeaders = true
	}
}

// MCPTool represents a tool definition in the ai-mcp-proxy plugin
type MCPTool struct {
	Description string                 `json:"description"`
	Method      string                 `json:"method,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Parameters  interface{}            `json:"parameters,omitempty"`
	Headers     map[string]interface{} `json:"headers,omitempty"`
	Query       map[string]interface{} `json:"query,omitempty"`
	Host        string                 `json:"host,omitempty"`
	Scheme      string                 `json:"scheme,omitempty"`
	RequestBody interface{}            `json:"request_body,omitempty"`
	Annotations *MCPAnnotations        `json:"annotations,omitempty"`
}

// MCPAnnotations represents the annotations for an MCP tool
type MCPAnnotations struct {
	Title            string `json:"title,omitempty"`
	ReadOnlyHint     bool   `json:"read_only_hint,omitempty"`
	IdempotentHint   bool   `json:"idempotent_hint,omitempty"`
	DestructiveHint  bool   `json:"destructive_hint,omitempty"`
	OpenWorldHint    bool   `json:"open_world_hint,omitempty"`
}

// MCPPluginConfig represents the configuration for the ai-mcp-proxy plugin
type MCPPluginConfig struct {
	Name    string                 `json:"name"`
	Route   string                 `json:"route,omitempty"`
	Service string                 `json:"service,omitempty"`
	Config  MCPPluginConfigDetails `json:"config"`
}

// MCPPluginConfigDetails represents the detailed configuration
type MCPPluginConfigDetails struct {
	Mode    string                 `json:"mode"`
	Tools   []MCPTool              `json:"tools"`
	Server  MCPServerConfig        `json:"server,omitempty"`
	Logging *MCPLoggingConfig      `json:"logging,omitempty"`
}

// MCPServerConfig represents the server configuration
type MCPServerConfig struct {
	Timeout              int    `json:"timeout,omitempty"`
	ForwardClientHeaders bool   `json:"forward_client_headers,omitempty"`
	Tag                  string `json:"tag,omitempty"`
}

// MCPLoggingConfig represents the logging configuration
type MCPLoggingConfig struct {
	LogPayloads   bool `json:"log_payloads,omitempty"`
	LogStatistics bool `json:"log_statistics,omitempty"`
}

// extractParameters converts OAS parameters to MCP tool parameters
func extractParameters(params []*v3.Parameter) (interface{}, error) {
	if len(params) == 0 {
		return nil, nil
	}

	parameters := make([]map[string]interface{}, 0)
	for _, param := range params {
		if param.In == "path" || param.In == "query" {
			paramDef := map[string]interface{}{
				"name":        param.Name,
				"in":          param.In,
				"required":    param.Required,
				"description": param.Description,
			}

			// Add schema if available
			if param.Schema != nil && param.Schema.Schema() != nil {
				schema := param.Schema.Schema()
				schemaDef := make(map[string]interface{})

				if len(schema.Type) > 0 {
					schemaDef["type"] = schema.Type[0]
				}
				if schema.Format != "" {
					schemaDef["format"] = schema.Format
				}
				if schema.Default != nil {
					schemaDef["default"] = schema.Default.Value
				}
				if len(schema.Enum) > 0 {
					enumValues := make([]interface{}, len(schema.Enum))
					for i, e := range schema.Enum {
						enumValues[i] = e.Value
					}
					schemaDef["enum"] = enumValues
				}

				paramDef["schema"] = schemaDef
			}

			parameters = append(parameters, paramDef)
		}
	}

	if len(parameters) == 0 {
		return nil, nil
	}

	// Return as JSON
	return parameters, nil
}

// extractRequestBody converts OAS requestBody to MCP tool request_body
func extractRequestBody(requestBody *v3.RequestBody) (interface{}, error) {
	if requestBody == nil || requestBody.Content == nil {
		return nil, nil
	}

	content := make(map[string]interface{})

	for pair := requestBody.Content.First(); pair != nil; pair = pair.Next() {
		mediaType := pair.Key()
		mediaTypeObj := pair.Value()

		if mediaTypeObj.Schema != nil && mediaTypeObj.Schema.Schema() != nil {
			schema := mediaTypeObj.Schema.Schema()
			schemaDef := buildSchemaDefinition(schema)

			if content["content"] == nil {
				content["content"] = make(map[string]interface{})
			}
			contentMap := content["content"].(map[string]interface{})
			contentMap[mediaType] = map[string]interface{}{
				"schema": schemaDef,
			}
		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	return content, nil
}

// buildSchemaDefinition recursively builds a schema definition
func buildSchemaDefinition(schema *openapibase.Schema) map[string]interface{} {
	schemaDef := make(map[string]interface{})

	if len(schema.Type) > 0 {
		schemaDef["type"] = schema.Type[0]
	}
	if schema.Format != "" {
		schemaDef["format"] = schema.Format
	}
	if schema.Description != "" {
		schemaDef["description"] = schema.Description
	}

	// Handle properties for object types
	if schema.Properties != nil && schema.Properties.Len() > 0 {
		properties := make(map[string]interface{})
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			propName := pair.Key()
			propSchema := pair.Value()
			if propSchema != nil && propSchema.Schema() != nil {
				properties[propName] = buildSchemaDefinition(propSchema.Schema())
			}
		}
		schemaDef["properties"] = properties
	}

	// Handle array items
	if schema.Items != nil && schema.Items.IsA() && schema.Items.A.Schema() != nil {
		schemaDef["items"] = buildSchemaDefinition(schema.Items.A.Schema())
	}

	// Handle required fields
	if len(schema.Required) > 0 {
		schemaDef["required"] = schema.Required
	}

	return schemaDef
}

// determineAnnotations determines MCP annotations based on HTTP method and operationId
// hasMCPKongTag checks if the operation has the "mcp:kong" tag
// extractMCPTags extracts all tags starting with "mcp:" from the operation tags
// Returns the tag values after the "mcp:" prefix
func extractMCPTags(tags []string) []string {
	mcpTags := make([]string, 0)
	for _, tag := range tags {
		if strings.HasPrefix(tag, "mcp:") {
			serverTag := strings.TrimPrefix(tag, "mcp:")
			if serverTag != "" {
				mcpTags = append(mcpTags, serverTag)
			}
		}
	}
	return mcpTags
}

func determineAnnotations(method string, operationId string) *MCPAnnotations {
	annotations := &MCPAnnotations{}

	// Set title if operationId is available
	if operationId != "" {
		annotations.Title = operationId
	}

	// Set hints based on HTTP method
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "OPTIONS":
		annotations.ReadOnlyHint = true
	case "PUT":
		annotations.IdempotentHint = true
	case "PATCH":
		// PATCH can be idempotent but also potentially destructive
		annotations.IdempotentHint = true
	case "DELETE":
		annotations.DestructiveHint = true
	case "POST":
		// POST can be various things, set open world hint
		annotations.OpenWorldHint = true
	}

	return annotations
}

// MustConvert is the same as Convert, but will panic if an error is returned.
func MustConvert(content []byte, opts O2MCPOptions) map[string]interface{} {
	result, err := Convert(content, opts)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// Convert converts an OpenAPI spec to an ai-mcp-proxy plugin configuration
// Convert converts an OpenAPI spec to ai-mcp-proxy plugin configurations
// Returns a map where keys are server tags and values are plugin configurations
func Convert(content []byte, opts O2MCPOptions) (map[string]interface{}, error) {
	opts.setDefaults()
	logbasics.Debug("received OpenAPI2MCP options", "options", opts)

	// Validate that route and service are not both specified
	if opts.RouteName != "" && opts.ServiceName != "" {
		return nil, fmt.Errorf("cannot specify both RouteName and ServiceName; they are mutually exclusive")
	}

	var doc v3.Document

	// Load and parse the OAS file
	openapiDoc, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing OAS3 file: [%w]", err)
	}

	// Check if circular references must be ignored
	if opts.IgnoreCircularRefs {
		docConfig := datamodel.NewDocumentConfiguration()
		docConfig.IgnoreArrayCircularReferences = true
		docConfig.IgnorePolymorphicCircularReferences = true
		openapiDoc.SetConfiguration(docConfig)
	}

	// Build the v3 model
	v3Model, errs := openapiDoc.BuildV3Model()
	if len(errs) > 0 {
		for i := range errs {
			logbasics.Error(errs[i], "error while building v3 document model \n")
		}
		return nil, fmt.Errorf("cannot create v3 model from document: %d errors reported", len(errs))
	}

	if v3Model != nil {
		doc = v3Model.Model
	}

	if doc.Paths == nil {
		return nil, fmt.Errorf("must have `.paths` in the root of the document")
	}

	// Group tools by server tag
	toolsByServerTag := make(map[string][]MCPTool)

	// Create a sorted array of paths for deterministic output
	allPaths := doc.Paths.PathItems
	sortedPaths := make([]string, allPaths.Len())
	path := allPaths.First()
	i := 0
	for path != nil && i < allPaths.Len() {
		sortedPaths[i] = path.Key()
		i++
		path = path.Next()
	}
	sort.Strings(sortedPaths)

	for _, pathKey := range sortedPaths {
		pathitem, ok := allPaths.Get(pathKey)
		if !ok {
			continue
		}

		logbasics.Info("processing path", "path", pathKey)

		// Get operations for this path
		operations := pathitem.GetOperations()

		// Create sorted array of methods
		sortedMethods := make([]string, operations.Len())
		method := operations.First()
		j := 0
		for method != nil && j < operations.Len() {
			sortedMethods[j] = method.Key()
			j++
			method = method.Next()
		}
		sort.Strings(sortedMethods)

		// Process each operation
		for _, methodKey := range sortedMethods {
			operation, ok := operations.Get(methodKey)
			if !ok {
				continue
			}

			methodKey = strings.ToUpper(methodKey)
			logbasics.Info("processing operation", "method", methodKey, "path", pathKey, "tags", operation.Tags)

			// Extract MCP tags
			mcpTags := extractMCPTags(operation.Tags)
			if len(mcpTags) == 0 {
				logbasics.Debug("skipping operation without mcp: tag", "method", methodKey, "path", pathKey)
				continue
			}

			tool := MCPTool{
				Method: methodKey,
				Path:   pathKey,
			}

			// Set description from operation summary or description
			if operation.Summary != "" {
				tool.Description = operation.Summary
			} else if operation.Description != "" {
				tool.Description = operation.Description
			} else {
				tool.Description = fmt.Sprintf("%s %s", methodKey, pathKey)
			}

			// Merge path-level and operation-level parameters
			allParams := make([]*v3.Parameter, 0)
			if pathitem.Parameters != nil {
				allParams = append(allParams, pathitem.Parameters...)
			}
			if operation.Parameters != nil {
				// Operation parameters override path parameters
				paramMap := make(map[string]*v3.Parameter)
				for _, p := range pathitem.Parameters {
					paramMap[p.Name+p.In] = p
				}
				for _, p := range operation.Parameters {
					paramMap[p.Name+p.In] = p
				}
				allParams = make([]*v3.Parameter, 0, len(paramMap))
				for _, p := range paramMap {
					allParams = append(allParams, p)
				}
			}

			// Extract parameters
			if len(allParams) > 0 {
				params, err := extractParameters(allParams)
				if err != nil {
					return nil, fmt.Errorf("failed to extract parameters for %s %s: %w", methodKey, pathKey, err)
				}
				tool.Parameters = params
			}

			// Extract request body
			if operation.RequestBody != nil {
				reqBody, err := extractRequestBody(operation.RequestBody)
				if err != nil {
					return nil, fmt.Errorf("failed to extract request body for %s %s: %w", methodKey, pathKey, err)
				}
				tool.RequestBody = reqBody
			}

			// Determine annotations based on method and operationId
			tool.Annotations = determineAnnotations(methodKey, operation.OperationId)

			// Add tool to each server tag
			for _, serverTag := range mcpTags {
				if toolsByServerTag[serverTag] == nil {
					toolsByServerTag[serverTag] = make([]MCPTool, 0)
				}
				toolsByServerTag[serverTag] = append(toolsByServerTag[serverTag], tool)
			}
		}
	}

	// Build plugin configurations for each server tag
	results := make(map[string]map[string]interface{})

	// Sort server tags for deterministic output
	serverTags := make([]string, 0, len(toolsByServerTag))
	for tag := range toolsByServerTag {
		serverTags = append(serverTags, tag)
	}
	sort.Strings(serverTags)

	for _, serverTag := range serverTags {
		tools := toolsByServerTag[serverTag]

		// Build the plugin configuration
		pluginConfig := MCPPluginConfig{
			Name:    "ai-mcp-proxy",
			Route:   opts.RouteName,
			Service: opts.ServiceName,
			Config: MCPPluginConfigDetails{
				Mode:  opts.Mode,
				Tools: tools,
				Server: MCPServerConfig{
					Timeout:              opts.ServerTimeout,
					ForwardClientHeaders: opts.ForwardClientHeaders,
					Tag:                  serverTag,
				},
			},
		}

		// Add logging config if needed
		if opts.LogPayloads || opts.LogStatistics {
			pluginConfig.Config.Logging = &MCPLoggingConfig{
				LogPayloads:   opts.LogPayloads,
				LogStatistics: opts.LogStatistics,
			}
		}

		// Convert to map for output
		jsonData, err := json.Marshal(pluginConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal plugin config for tag %s: %w", serverTag, err)
		}

		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin config for tag %s: %w", serverTag, err)
		}

		// Remove route or service if empty
		if opts.RouteName == "" {
			delete(result, "route")
		}
		if opts.ServiceName == "" {
			delete(result, "service")
		}

		results[serverTag] = result
	}

	// Assemble all plugins into a single Kong declarative format
	allPlugins := make([]interface{}, 0, len(results))
	
	// Sort server tags for deterministic output
	sortedTags := make([]string, 0, len(results))
	for tag := range results {
		sortedTags = append(sortedTags, tag)
	}
	sort.Strings(sortedTags)
	
	// Add all plugins to the array
	for _, tag := range sortedTags {
		allPlugins = append(allPlugins, results[tag])
	}
	
	// Create final output with Kong declarative format
	output := map[string]interface{}{
		"_format_version": "3.0",
		"plugins":         allPlugins,
	}

	logbasics.Debug("finished processing document", "server_tags", len(results))
	return output, nil
}
