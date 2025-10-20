Configuration
configobjectrequired
Hide Child Parameters
loggingobject
Hide Child Parameters
log_payloadsboolean
If enabled, will log the request and response body into the Kong log plugin(s) output.

Default:false

log_statisticsboolean
If enabled, will add mcp metrics into the Kong log plugin(s) output.

Default:false

max_request_body_sizeinteger
max allowed body size allowed to be handled as MCP request.

Default:8192

modestringrequired
The mode of the MCP proxy. Possible values are: ‘passthrough-listener’, ‘conversion-listener’, ‘conversion-only’, ‘listener’.

Allowed values:
conversion-listener
conversion-only
listener
passthrough-listener

serverobject
Hide Child Parameters
forward_client_headersboolean
Whether to forward the client request headers to the upstream server when calling the tools.

Default:true

tagstring
The tag of the MCP server. This is used to filter the exported MCP tools. The field should contain exactly one tag.

timeoutnumber
The timeout for calling the tools in milliseconds.

Default:10000

toolsarray[object]
Hide Child Parameters
annotationsobject
Hide Child Parameters
destructive_hintboolean
If true, the tool may perform destructive updates

idempotent_hintboolean
If true, repeated calls with same args have no additional effect

open_world_hintboolean
If true, tool interacts with external entities

read_only_hintboolean
If true, the tool does not modify its environment

titlestring
Human-readable title for the tool

descriptionstringrequired
The description of the MCP tool. This is used to provide information about the tool’s functionality and usage.

headersobject
The headers of the exported API. By default, Kong will extract the headers from API configuration. If the configured headers are not exactly matched, this field is required.

* Additional properties are allowed.
hoststring
The host of the exported API. By default, Kong will extract the host from API configuration. If the configured host is wildcard, this field is required.

methodstring
The method of the exported API. By default, Kong will extract the method from API configuration. If the configured method is not exactly matched, this field is required.

Allowed values:
DELETE
GET
PATCH
POST
PUT

parametersjson
The API parameters specification defined in OpenAPI. For example, ‘[{“name”: “city”, “in”: “query”, “description”: “Name of the city to get the weather for”, “required”: true, “schema”: {“type”: “string”}}]’.See https://swagger.io/docs/specification/v3_0/describing-parameters/ for more details.

pathstring
The path of the exported API. By default, Kong will extract the path from API configuration. If the configured path is not exactly matched, this field is required. Paths not starting with ‘/’ are treated as relative paths.

queryobject
The query arguments of the exported API. If the generated query arguments are not exactly matched, this field is required.

* Additional properties are allowed.
request_bodyjson
The API requestBody specification defined in OpenAPI. For example, ‘{“content”:{“application/x-www-form-urlencoded”:{“schema”:{“type”:“object”,“properties”:{“color”:{“type”:“array”,“items”:{“type”:“string”}}}}}}’.See https://swagger.io/docs/specification/v3_0/describing-request-body/describing-request-body/ for more details.

schemestring
The scheme of the exported API. By default, Kong will extract the scheme from API configuration. If the configured scheme is not expected, this field can be used to override it.

Allowed values:
http
https

protocolsarray[string]
A set of strings representing HTTP protocols.

Allowed values:
grpc
grpcs
http
https

Default:grpc, grpcs, http, https

routeobject
If set, the plugin will only activate when receiving requests via the specified route. Leave unset for the plugin to activate regardless of the route being used.

* Additional properties are NOT allowed.
Hide Child Parameters
idstring
serviceobject
If set, the plugin will only activate when receiving requests via one of the routes belonging to the specified Service. Leave unset for the plugin to activate regardless of the Service being matched.

* Additional properties are NOT allowed.
Hide Child Parameters
idstring
