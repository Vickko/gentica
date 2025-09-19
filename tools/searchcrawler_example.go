package tools

// Example usage with genkit adapter:
//
// func RegisterAllTools(g *genkit.Genkit, workingDir string) []ai.Tool {
//     existingTools := []BaseTool{
//         NewViewTool(workingDir),
//         NewEditTool(workingDir),
//         NewWriteTool(workingDir),
//         NewGlobTool(workingDir),
//         NewGrepTool(workingDir),
//         NewSearchCrawlerTool(workingDir),  // Add the new SearchCrawler tool
//         // ... other tools
//     }
//
//     return BatchAdaptTools(g, existingTools...)
// }

// Example direct usage:
//
// import (
//     "context"
//     "encoding/json"
//     "fmt"
//     "gentica/tools"
// )
//
// func main() {
//     // Create the tool
//     tool := tools.NewSearchCrawlerTool("/path/to/working/dir")
//
//     // Prepare parameters
//     params := tools.SearchCrawlerParams{
//         Query:     "golang best practices",
//         ResultNum: 5,
//         SavePath:  "./research/golang",
//     }
//
//     // Marshal parameters to JSON
//     paramsJSON, _ := json.Marshal(params)
//
//     // Create tool call
//     call := tools.ToolCall{
//         Name:  "searchCrawler",
//         Input: string(paramsJSON),
//     }
//
//     // Execute the tool
//     response, err := tool.Run(context.Background(), call)
//     if err != nil {
//         fmt.Printf("Error: %v\n", err)
//         return
//     }
//
//     // Check response
//     if response.IsError {
//         fmt.Printf("Tool error: %s\n", response.Content)
//     } else {
//         fmt.Printf("Result: %s\n", response.Content)
//     }
// }