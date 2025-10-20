package openapi2mcp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kong/go-apiops/openapi2mcp"
)

// Helper function to get plugin by server tag from result
func getPluginByTag(result map[string]interface{}, tag string) map[string]interface{} {
	plugins := result["plugins"].([]interface{})
	for _, p := range plugins {
		plugin := p.(map[string]interface{})
		config := plugin["config"].(map[string]interface{})
		server := config["server"].(map[string]interface{})
		if server["tag"] == tag {
			return plugin
		}
	}
	return nil
}

var _ = Describe("openapi2mcp", func() {

	Describe("Convert", func() {
		Context("with a simple OpenAPI spec and single server tag", func() {
			It("should convert to MCP plugin configuration", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get users
      operationId: getUsers
      tags:
        - mcp:users
      responses:
        '200':
          description: Success
  /users/{id}:
    get:
      summary: Get user by id
      operationId: getUserById
      tags:
        - mcp:users
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
          description: User ID
      responses:
        '200':
          description: Success
    delete:
      summary: Delete user
      operationId: deleteUser
      tags:
        - mcp:users
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: Deleted
`)

				opts := openapi2mcp.O2MCPOptions{
					RouteName:     "test-route",
					Mode:          "conversion-listener",
					ServerTimeout: 60000,
					LogStatistics: true,
				}

				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())

				// Verify format version
				Expect(result["_format_version"]).To(Equal("3.0"))

				// Verify plugins array
				plugins := result["plugins"].([]interface{})
				Expect(len(plugins)).To(Equal(1))

				// Get the plugin
				plugin := plugins[0].(map[string]interface{})

				// Verify plugin name
				Expect(plugin["name"]).To(Equal("ai-mcp-proxy"))
				Expect(plugin["route"]).To(Equal("test-route"))
				// Service should not be present when route is specified
				Expect(plugin["service"]).To(BeNil())

				// Verify config
				config, ok := plugin["config"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(config["mode"]).To(Equal("conversion-listener"))

				// Verify tools
				tools, ok := config["tools"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(len(tools)).To(Equal(3)) // GET /users, DELETE /users/{id}, GET /users/{id}

				// Tools are sorted by path first, then by method within each path
				// First tool: GET /users
				tool1 := tools[0].(map[string]interface{})
				Expect(tool1["method"]).To(Equal("GET"))
				Expect(tool1["path"]).To(Equal("/users"))
				Expect(tool1["description"]).To(Equal("Get users"))

				// Check annotations for GET
				annotations1 := tool1["annotations"].(map[string]interface{})
				Expect(annotations1["read_only_hint"]).To(Equal(true))

				// Second tool: DELETE /users/{id}
				tool2 := tools[1].(map[string]interface{})
				Expect(tool2["method"]).To(Equal("DELETE"))
				Expect(tool2["path"]).To(Equal("/users/{id}"))
				Expect(tool2["description"]).To(Equal("Delete user"))

				// Check annotations for DELETE
				annotations2 := tool2["annotations"].(map[string]interface{})
				Expect(annotations2["destructive_hint"]).To(Equal(true))

				// Verify server config with server tag
				server, ok := config["server"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(server["timeout"]).To(Equal(float64(60000)))

				// Verify server tag is set
				tag := server["tag"]
				Expect(tag).To(Equal("users"))

				// Verify logging config
				logging, ok := config["logging"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(logging["log_statistics"]).To(Equal(true))
			})
		})

		Context("with multiple server tags", func() {
			It("should generate separate configs for each server tag", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /flights:
    get:
      summary: Get flights
      operationId: getFlights
      tags:
        - mcp:flights
        - mcp:monitoring
      responses:
        '200':
          description: Success
  /hotels:
    get:
      summary: Get hotels
      operationId: getHotels
      tags:
        - mcp:hotels
      responses:
        '200':
          description: Success
  /health:
    get:
      summary: Health check
      operationId: healthCheck
      tags:
        - mcp:monitoring
      responses:
        '200':
          description: Success
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				// Verify format version
				Expect(result["_format_version"]).To(Equal("3.0"))

				// Should have three plugins (one per server tag)
				plugins := result["plugins"].([]interface{})
				Expect(len(plugins)).To(Equal(3))

				// Check flights config
				flightsConfig := getPluginByTag(result, "flights")
				Expect(flightsConfig).NotTo(BeNil())
				flightsTools := flightsConfig["config"].(map[string]interface{})["tools"].([]interface{})
				Expect(len(flightsTools)).To(Equal(1))
				Expect(flightsTools[0].(map[string]interface{})["path"]).To(Equal("/flights"))

				// Check hotels config
				hotelsConfig := getPluginByTag(result, "hotels")
				Expect(hotelsConfig).NotTo(BeNil())
				hotelsTools := hotelsConfig["config"].(map[string]interface{})["tools"].([]interface{})
				Expect(len(hotelsTools)).To(Equal(1))
				Expect(hotelsTools[0].(map[string]interface{})["path"]).To(Equal("/hotels"))

				// Check monitoring config - should have both /flights and /health
				monitoringConfig := getPluginByTag(result, "monitoring")
				Expect(monitoringConfig).NotTo(BeNil())
				monitoringTools := monitoringConfig["config"].(map[string]interface{})["tools"].([]interface{})
				Expect(len(monitoringTools)).To(Equal(2))

				// Verify server tags
				flightsServer := flightsConfig["config"].(map[string]interface{})["server"].(map[string]interface{})
				Expect(flightsServer["tag"]).To(Equal("flights"))

				hotelsServer := hotelsConfig["config"].(map[string]interface{})["server"].(map[string]interface{})
				Expect(hotelsServer["tag"]).To(Equal("hotels"))

				monitoringServer := monitoringConfig["config"].(map[string]interface{})["server"].(map[string]interface{})
				Expect(monitoringServer["tag"]).To(Equal("monitoring"))
			})
		})

		Context("with parameters", func() {
			It("should extract path and query parameters", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items/{id}:
    get:
      summary: Get item
      tags:
        - mcp:items
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: fields
          in: query
          required: false
          schema:
            type: string
          description: Comma-separated list of fields
      responses:
        '200':
          description: Success
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["_format_version"]).To(Equal("3.0"))
				plugin := getPluginByTag(result, "items")
				Expect(plugin).NotTo(BeNil())
				config := plugin["config"].(map[string]interface{})
				tools := config["tools"].([]interface{})
				Expect(len(tools)).To(Equal(1))

				tool := tools[0].(map[string]interface{})
				params := tool["parameters"].([]interface{})
				Expect(len(params)).To(Equal(2))

				// Check path parameter
				param1 := params[0].(map[string]interface{})
				Expect(param1["name"]).To(Equal("id"))
				Expect(param1["in"]).To(Equal("path"))
				Expect(param1["required"]).To(Equal(true))

				// Check query parameter
				param2 := params[1].(map[string]interface{})
				Expect(param2["name"]).To(Equal("fields"))
				Expect(param2["in"]).To(Equal("query"))
				Expect(param2["required"]).To(Equal(false))
			})
		})

		Context("with request body", func() {
			It("should extract request body schema", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    post:
      summary: Create user
      tags:
        - mcp:users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                email:
                  type: string
              required:
                - name
      responses:
        '201':
          description: Created
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["_format_version"]).To(Equal("3.0"))
				plugin := getPluginByTag(result, "users")
				Expect(plugin).NotTo(BeNil())
				config := plugin["config"].(map[string]interface{})
				tools := config["tools"].([]interface{})
				Expect(len(tools)).To(Equal(1))

				tool := tools[0].(map[string]interface{})
				Expect(tool["method"]).To(Equal("POST"))

				// Check request body
				requestBody := tool["request_body"].(map[string]interface{})
				Expect(requestBody).NotTo(BeNil())

				content := requestBody["content"].(map[string]interface{})
				appJson := content["application/json"].(map[string]interface{})
				schema := appJson["schema"].(map[string]interface{})

				Expect(schema["type"]).To(Equal("object"))
				properties := schema["properties"].(map[string]interface{})
				Expect(properties["name"]).NotTo(BeNil())
				Expect(properties["email"]).NotTo(BeNil())
			})
		})

		Context("with default options", func() {
			It("should set default values", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      tags:
        - mcp:test
      responses:
        '200':
          description: Success
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["_format_version"]).To(Equal("3.0"))
				plugin := getPluginByTag(result, "test")
				Expect(plugin).NotTo(BeNil())
				// Route and service should not be present when not specified
				Expect(plugin["route"]).To(BeNil())
				Expect(plugin["service"]).To(BeNil())
				config := plugin["config"].(map[string]interface{})
				Expect(config["mode"]).To(Equal("conversion-listener"))

				server := config["server"].(map[string]interface{})
				Expect(server["timeout"]).To(Equal(float64(60000)))
				Expect(server["forward_client_headers"]).To(Equal(true))
			})
		})

		Context("with invalid spec", func() {
			It("should return error for invalid YAML", func() {
				spec := []byte(`invalid: yaml: content: [`)

				opts := openapi2mcp.O2MCPOptions{}
				_, err := openapi2mcp.Convert(spec, opts)
				Expect(err).To(HaveOccurred())
			})

			It("should return error for spec without paths", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
`)

				opts := openapi2mcp.O2MCPOptions{}
				_, err := openapi2mcp.Convert(spec, opts)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have `.paths`"))
			})
		})

		Context("with mcp: tag filtering", func() {
			It("should only include operations with mcp: tags", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /flights:
    get:
      summary: Get flights
      operationId: getFlights
      tags:
        - mcp:flights
        - other-tag
      responses:
        '200':
          description: Success
  /bookings:
    get:
      summary: Get bookings
      operationId: getBookings
      tags:
        - no-mcp-tag
      responses:
        '200':
          description: Success
  /hotels:
    get:
      summary: Get hotels
      operationId: getHotels
      tags:
        - mcp:hotels
      responses:
        '200':
          description: Success
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["_format_version"]).To(Equal("3.0"))
				plugins := result["plugins"].([]interface{})
				
				// Should have 2 plugins (flights and hotels), bookings should be excluded
				Expect(len(plugins)).To(Equal(2))

				// Check flights config exists
				flightsPlugin := getPluginByTag(result, "flights")
				Expect(flightsPlugin).NotTo(BeNil())

				// Check hotels config exists
				hotelsPlugin := getPluginByTag(result, "hotels")
				Expect(hotelsPlugin).NotTo(BeNil())

				// bookings should not be present
				bookingsPlugin := getPluginByTag(result, "bookings")
				Expect(bookingsPlugin).To(BeNil())
			})

			It("should return empty map when no mcp: tags present", func() {
				spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      summary: Test endpoint
      tags:
        - regular-tag
      responses:
        '200':
          description: Success
`)

				opts := openapi2mcp.O2MCPOptions{}
				result, err := openapi2mcp.Convert(spec, opts)
				Expect(err).NotTo(HaveOccurred())

				Expect(result["_format_version"]).To(Equal("3.0"))
				plugins := result["plugins"].([]interface{})
				// Should have no plugins
				Expect(len(plugins)).To(Equal(0))
			})
		})
	})

	Describe("MustConvert", func() {
		// Note: MustConvert calls log.Fatal which exits the process, so we can't test panics directly
		// We only test the success case
		It("should not panic on success", func() {
			spec := []byte(`
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      tags:
        - mcp:test
      responses:
        '200':
          description: Success
`)
			opts := openapi2mcp.O2MCPOptions{}

			result := openapi2mcp.MustConvert(spec, opts)
			Expect(result).NotTo(BeNil())
			Expect(result["_format_version"]).To(Equal("3.0"))
			plugins := result["plugins"].([]interface{})
			Expect(len(plugins)).To(Equal(1))
		})
	})
})
