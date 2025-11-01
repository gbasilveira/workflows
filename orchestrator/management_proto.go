package orchestrator

// This file will contain proto conversion functions after proto files are generated.
//
// To complete the implementation:
//
// 1. Run ./generate-proto.sh to generate proto code
//
// 2. Update imports:
//    import proto "github.com/gbasilveira/dag-engine/orchestrator/proto/gen"
//
// 3. Implement workflowDefinitionToProto:
//    func workflowDefinitionToProto(def *WorkflowDefinition) *proto.WorkflowDefinition {
//        protoNodes := make([]*proto.NodeDefinition, 0, len(def.Nodes))
//        for _, node := range def.Nodes {
//            executorConfig := make(map[string]string)
//            for k, v := range node.ExecutorConfig {
//                executorConfig[k] = fmt.Sprintf("%v", v)
//            }
//            
//            nodeMetadata := make(map[string]string)
//            for k, v := range node.Metadata {
//                nodeMetadata[k] = fmt.Sprintf("%v", v)
//            }
//
//            protoNode := &proto.NodeDefinition{
//                NodeId:        node.NodeID,
//                Dependencies:  node.Dependencies,
//                ExecutorType:  node.ExecutorType,
//                ExecutorCode:  node.ExecutorCode,
//                ExecutorConfig: executorConfig,
//                Metadata:      nodeMetadata,
//            }
//            protoNodes = append(protoNodes, protoNode)
//        }
//
//        protoMetadata := make(map[string]string)
//        for k, v := range def.Metadata {
//            protoMetadata[k] = fmt.Sprintf("%v", v)
//        }
//
//        return &proto.WorkflowDefinition{
//            WorkflowId: def.WorkflowID,
//            Version:    def.Version,
//            Name:      def.Name,
//            Nodes:     protoNodes,
//            Metadata:  protoMetadata,
//            CreatedAt: time.Now().Unix(),
//            UpdatedAt: time.Now().Unix(),
//        }
//    }
//
// 4. Implement protoToWorkflowDefinition:
//    func protoToWorkflowDefinition(protoDef *proto.WorkflowDefinition) (*WorkflowDefinition, error) {
//        nodes := make([]NodeDefinition, 0, len(protoDef.Nodes))
//        for _, protoNode := range protoDef.Nodes {
//            executorConfig := make(map[string]interface{})
//            for k, v := range protoNode.ExecutorConfig {
//                executorConfig[k] = v
//            }
//
//            metadata := make(map[string]interface{})
//            for k, v := range protoNode.Metadata {
//                metadata[k] = v
//            }
//
//            node := NodeDefinition{
//                NodeID:       protoNode.NodeId,
//                Dependencies: protoNode.Dependencies,
//                ExecutorType: protoNode.ExecutorType,
//                ExecutorCode: protoNode.ExecutorCode,
//                ExecutorConfig: executorConfig,
//                Metadata:     metadata,
//            }
//            nodes = append(nodes, node)
//        }
//
//        metadata := make(map[string]interface{})
//        for k, v := range protoDef.Metadata {
//            metadata[k] = v
//        }
//
//        return &WorkflowDefinition{
//            WorkflowID: protoDef.WorkflowId,
//            Version:    protoDef.Version,
//            Name:      protoDef.Name,
//            Nodes:     nodes,
//            Metadata:  metadata,
//        }, nil
//    }

