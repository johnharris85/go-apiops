package cmd

import (
	"fmt"
	"log"

	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/openapi2mcp"
	"github.com/spf13/cobra"
)

// Executes the CLI command "openapi2mcp"
func executeOpenapi2MCP(cmd *cobra.Command, _ []string) error {
	verbosity, _ := cmd.Flags().GetInt("verbose")
	logbasics.Initialize(log.LstdFlags, verbosity)

	inputFilename, err := cmd.Flags().GetString("spec")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'spec'; %w", err)
	}

	outputFilename, err := cmd.Flags().GetString("output-file")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'output-file'; %w", err)
	}

	routeName, err := cmd.Flags().GetString("route-name")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'route-name'; %w", err)
	}

	serviceName, err := cmd.Flags().GetString("service-name")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'service-name'; %w", err)
	}

	// Validate that route-name and service-name are not both specified
	if routeName != "" && serviceName != "" {
		return fmt.Errorf("cannot specify both --route-name and --service-name; they are mutually exclusive")
	}

	pathPrefix, err := cmd.Flags().GetString("path-prefix")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'path-prefix'; %w", err)
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'mode'; %w", err)
	}

	serverTimeout, err := cmd.Flags().GetInt("server-timeout")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'server-timeout'; %w", err)
	}

	logStatistics, err := cmd.Flags().GetBool("log-statistics")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'log-statistics'; %w", err)
	}

	logPayloads, err := cmd.Flags().GetBool("log-payloads")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'log-payloads'; %w", err)
	}

	forwardClientHeaders, err := cmd.Flags().GetBool("forward-client-headers")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'forward-client-headers'; %w", err)
	}

	ignoreCircularRefs, err := cmd.Flags().GetBool("ignore-circular-refs")
	if err != nil {
		return fmt.Errorf("failed getting cli argument 'ignore-circular-refs'; %w", err)
	}

	var outputFormat string
	{
		outputFormat, err = cmd.Flags().GetString("format")
		if err != nil {
			return fmt.Errorf("failed getting cli argument 'format'; %w", err)
		}
	}

	options := openapi2mcp.O2MCPOptions{
		RouteName:            routeName,
		ServiceName:          serviceName,
		PathPrefix:           pathPrefix,
		Mode:                 mode,
		ServerTimeout:        serverTimeout,
		LogStatistics:        logStatistics,
		LogPayloads:          logPayloads,
		ForwardClientHeaders: forwardClientHeaders,
		IgnoreCircularRefs:   ignoreCircularRefs,
	}

	// do the work: read/convert/write
	content, err := filebasics.ReadFile(inputFilename)
	if err != nil {
		return err
	}

	result, err := openapi2mcp.Convert(content, options)
	if err != nil {
		return fmt.Errorf("failed converting OpenAPI spec '%s' to MCP plugin config; %w", inputFilename, err)
	}

	return filebasics.WriteSerializedFile(outputFilename, result, filebasics.OutputFormat(outputFormat))
}

//
//
// Define the CLI data for the openapi2mcp command
//
//

var openapi2mcpCmd = &cobra.Command{
	Use:   "openapi2mcp",
	Short: "Convert OpenAPI files to ai-mcp-proxy plugin configuration",
	Long: `Convert OpenAPI files to ai-mcp-proxy plugin configuration.

This command reads an OpenAPI specification and generates Kong ai-mcp-proxy
plugin configurations that expose the API operations as MCP (Model Context Protocol) tools.

Operations are filtered by tags starting with "mcp:" - only operations with such tags
will be included. The value after "mcp:" becomes the server tag. Operations can have
multiple mcp: tags, and separate plugin configurations will be generated for each server tag.

The generated configuration includes:
- Tool definitions for each API operation
- Parameter mappings (path, query, headers)
- Request body schemas
- Annotations based on HTTP methods (read-only, idempotent, destructive)

If multiple server tags are found, the output will be a map of server tags to plugin configs.
If only one server tag is found, the output will be a single plugin config.

Example:
  go-apiops openapi2mcp -s api-spec.yaml -o mcp-plugin.yaml`,
	RunE: executeOpenapi2MCP,
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(openapi2mcpCmd)
	openapi2mcpCmd.Flags().StringP("spec", "s", "-", "OpenAPI spec file to process. Use - to read from stdin")
	openapi2mcpCmd.Flags().StringP("output-file", "o", "-", "output file to write. Use - to write to stdout")
	openapi2mcpCmd.Flags().StringP("format", "", string(filebasics.OutputFormatYaml), "output format: "+
		string(filebasics.OutputFormatJSON)+" or "+string(filebasics.OutputFormatYaml))
	openapi2mcpCmd.Flags().StringP("route-name", "", "",
		"the route name/ID to associate with the ai-mcp-proxy plugin (mutually exclusive with --service-name)")
	openapi2mcpCmd.Flags().StringP("service-name", "", "",
		"the service name/ID to associate with the ai-mcp-proxy plugin (mutually exclusive with --route-name)")
	openapi2mcpCmd.Flags().StringP("path-prefix", "", "",
		"path prefix to prepend to all tool paths (e.g., /api/v1)")
	openapi2mcpCmd.Flags().StringP("mode", "", "conversion-listener",
		"the mode of the MCP proxy (conversion-listener, conversion-only, listener, passthrough-listener)")
	openapi2mcpCmd.Flags().IntP("server-timeout", "", 60000,
		"timeout for calling the tools in milliseconds")
	openapi2mcpCmd.Flags().BoolP("log-statistics", "", false,
		"enable logging of MCP metrics")
	openapi2mcpCmd.Flags().BoolP("log-payloads", "", false,
		"enable logging of request and response bodies")
	openapi2mcpCmd.Flags().BoolP("forward-client-headers", "", true,
		"forward client request headers to upstream server when calling tools")
	openapi2mcpCmd.Flags().BoolP("ignore-circular-refs", "", false,
		"ignore circular references in the spec")
}
