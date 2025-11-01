# Management Service

The Management Service provides a REST API for managing workflows via YAML definitions.

## Prerequisites

1. **Generate Proto Files**: Run `./generate-proto.sh` in the project root to generate gRPC code
2. **Complete Proto Integration**: Once proto files are generated, update the following files:
   - `orchestrator_client.go` - Wire proto client calls
   - `orchestrator/management.go` - Update to use proto types
   - `orchestrator/management_proto.go` - Implement conversion functions

## Building

```bash
go build -o bin/management ./cmd/management
```

## Running

```bash
./bin/management -port 8080 -orchestrator localhost:50051
```

## API Endpoints

### POST /api/v1/workflows
Upload a YAML workflow definition.

**Request Body**: YAML workflow spec
**Response**: 
```json
{
  "success": true,
  "workflow_id": "data-pipeline",
  "version": "1.0.0",
  "message": "Workflow registered successfully"
}
```

### PUT /api/v1/workflows/{id}
Update a workflow definition.

**Query Parameters**:
- `force=true` - Force update even if dependents exist

### DELETE /api/v1/workflows/{id}
Delete a workflow.

**Query Parameters**:
- `version=1.0.0` - Delete specific version (optional, deletes all if not specified)
- `force=true` - Force delete even if dependents exist

### GET /api/v1/workflows
List all workflows.

**Query Parameters**:
- `filter=<filter>` - Optional filter (future: label selector)

### GET /api/v1/workflows/{id}
Get a specific workflow.

**Query Parameters**:
- `version=1.0.0` - Get specific version (optional, gets latest if not specified)

### GET /health
Health check endpoint.

## Example Usage

```bash
# Register a workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/x-yaml" \
  --data-binary @examples/sample-workflow.yaml

# Get a workflow
curl http://localhost:8080/api/v1/workflows/data-pipeline

# List all workflows
curl http://localhost:8080/api/v1/workflows

# Update a workflow
curl -X PUT http://localhost:8080/api/v1/workflows/data-pipeline \
  -H "Content-Type: application/x-yaml" \
  --data-binary @examples/sample-workflow.yaml

# Delete a workflow
curl -X DELETE http://localhost:8080/api/v1/workflows/data-pipeline
```

## Next Steps After Proto Generation

1. Update `orchestrator_client.go` to use proto client
2. Uncomment proto imports in all files
3. Implement proto conversion functions in `orchestrator/management_proto.go`
4. Test the full integration

