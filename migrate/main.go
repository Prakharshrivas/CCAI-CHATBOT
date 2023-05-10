// Package main implements the ES to CX migration tool.
package main

import (
        "context"
        "encoding/csv"
        "flag"
        "fmt"
        "os"
        "strings"
        "time"

        v2 "cloud.google.com/go/dialogflow/apiv2"
        proto2 "cloud.google.com/go/dialogflow/apiv2/dialogflowpb"
        v3 "cloud.google.com/go/dialogflow/cx/apiv3"
        proto3 "cloud.google.com/go/dialogflow/cx/apiv3/cxpb"
        "google.golang.org/api/iterator"
        "google.golang.org/api/option"
)

// Commandline flags
var v2Project *string = flag.String("es-project-id", "", "ES project")
var v3Project *string = flag.String("cx-project-id", "", "CX project")
var v2Region *string = flag.String("es-region-id", "", "ES region")
var v3Region *string = flag.String("cx-region-id", "", "CX region")
var v3Agent *string = flag.String("cx-agent-id", "", "CX region")
var outFile *string = flag.String("out-file", "", "Output file for CSV TODO items")
var dryRun *bool = flag.Bool("dry-run", false, "Set true to skip CX agent writes")

// Map from entity type display name to fully qualified name.
var entityTypeShortToLong = map[string]string{}

// Map from ES system entity to CX system entity
var convertSystemEntity = map[string]string{
        "sys.address":         "sys.address",
        "sys.any":             "sys.any",
        "sys.cardinal":        "sys.cardinal",
        "sys.color":           "sys.color",
        "sys.currency-name":   "sys.currency-name",
        "sys.date":            "sys.date",
        "sys.date-period":     "sys.date-period",
        "sys.date-time":       "sys.date-time",
        "sys.duration":        "sys.duration",
        "sys.email":           "sys.email",
        "sys.flight-number":   "sys.flight-number",
        "sys.geo-city-gb":     "sys.geo-city",
        "sys.geo-city-us":     "sys.geo-city",
        "sys.geo-city":        "sys.geo-city",
        "sys.geo-country":     "sys.geo-country",
        "sys.geo-state":       "sys.geo-state",
        "sys.geo-state-us":    "sys.geo-state",
        "sys.geo-state-gb":    "sys.geo-state",
        "sys.given-name":      "sys.given-name",
        "sys.language":        "sys.language",
        "sys.last-name":       "sys.last-name",
        "sys.street-address":  "sys.location",
        "sys.location":        "sys.location",
        "sys.number":          "sys.number",
        "sys.number-integer":  "sys.number-integer",
        "sys.number-sequence": "sys.number-sequence",
        "sys.ordinal":         "sys.ordinal",
        "sys.percentage":      "sys.percentage",
        "sys.person":          "sys.person",
        "sys.phone-number":    "sys.phone-number",
        "sys.temperature":     "sys.temperature",
        "sys.time":            "sys.time",
        "sys.time-period":     "sys.time-period",
        "sys.unit-currency":   "sys.unit-currency",
        "sys.url":             "sys.url",
        "sys.zip-code":        "sys.zip-code",
}

// Issues found for the CSV output
var issues = [][]string{
        {"Field", "Issue"},
}

// logIssue logs an issue for the CSV output
func logIssue(field string, issue string) {
        issues = append(issues, []string{field, issue})
}

// convertEntityType converts an ES entity type to CX
func convertEntityType(et2 *proto2.EntityType) *proto3.EntityType {
        var kind3 proto3.EntityType_Kind
        switch kind2 := et2.Kind; kind2 {
        case proto2.EntityType_KIND_MAP:
                kind3 = proto3.EntityType_KIND_MAP
        case proto2.EntityType_KIND_LIST:
                kind3 = proto3.EntityType_KIND_LIST
        case proto2.EntityType_KIND_REGEXP:
                kind3 = proto3.EntityType_KIND_REGEXP
        default:
                kind3 = proto3.EntityType_KIND_UNSPECIFIED
        }
        var expansion3 proto3.EntityType_AutoExpansionMode
        switch expansion2 := et2.AutoExpansionMode; expansion2 {
        case proto2.EntityType_AUTO_EXPANSION_MODE_DEFAULT:
                expansion3 = proto3.EntityType_AUTO_EXPANSION_MODE_DEFAULT
        default:
                expansion3 = proto3.EntityType_AUTO_EXPANSION_MODE_UNSPECIFIED
        }
        et3 := &proto3.EntityType{
                DisplayName:           et2.DisplayName,
                Kind:                  kind3,
                AutoExpansionMode:     expansion3,
                EnableFuzzyExtraction: et2.EnableFuzzyExtraction,
        }
        for _, e2 := range et2.Entities {
                et3.Entities = append(et3.Entities, &proto3.EntityType_Entity{
                        Value:    e2.Value,
                        Synonyms: e2.Synonyms,
                })
        }
        return et3
}

// convertParameterEntityType converts a entity type found in parameters
func convertParameterEntityType(intent string, parameter string, t2 string) string {
        if len(t2) == 0 {
                return ""
        }
        t2 = t2[1:] // remove @
        if strings.HasPrefix(t2, "sys.") {
                if val, ok := convertSystemEntity[t2]; ok {
                        t2 = val
                } else {
                        t2 = "sys.any"
                        logIssue("Intent<"+intent+">.Parameter<"+parameter+">",
                                "This intent parameter uses a system entity not supported by CX English agents. See the migration guide for advice. System entity: "+t2)
                }
                return fmt.Sprintf("projects/-/locations/-/agents/-/entityTypes/%s", t2)
        }
        return entityTypeShortToLong[t2]
}

// convertIntent converts an ES intent to CX
func convertIntent(intent2 *proto2.Intent) *proto3.Intent {
        if intent2.DisplayName == "Default Fallback Intent" ||
                intent2.DisplayName == "Default Welcome Intent" {
                return nil
        }

        intent3 := &proto3.Intent{
                DisplayName: intent2.DisplayName,
        }

        // WebhookState
        if intent2.WebhookState != proto2.Intent_WEBHOOK_STATE_UNSPECIFIED {
                logIssue("Intent<"+intent2.DisplayName+">.WebhookState",
                        "This intent has webhook enabled. You must configure this in your CX agent.")
        }

        // IsFallback
        if intent2.IsFallback {
                logIssue("Intent<"+intent2.DisplayName+">.IsFallback",
                        "This intent is a fallback intent. CX does not support this. Use no-match events instead.")
        }

        // MlDisabled
        if intent2.MlDisabled {
                logIssue("Intent<"+intent2.DisplayName+">.MlDisabled",
                        "This intent has ML disabled. CX does not support this.")
        }

        // LiveAgentHandoff
        if intent2.LiveAgentHandoff {
                logIssue("Intent<"+intent2.DisplayName+">.LiveAgentHandoff",
                        "This intent uses live agent handoff. You must configure this in a fulfillment.")
        }

        // EndInteraction
        if intent2.EndInteraction {
                logIssue("Intent<"+intent2.DisplayName+">.EndInteraction",
                        "This intent uses end interaction. CX does not support this.")
        }

        // InputContextNames
        if len(intent2.InputContextNames) > 0 {
                logIssue("Intent<"+intent2.DisplayName+">.InputContextNames",
                        "This intent uses context. See the migration guide for alternatives.")
        }

        // Events
        if len(intent2.Events) > 0 {
                logIssue("Intent<"+intent2.DisplayName+">.Events",
                        "This intent uses events. Use event handlers instead.")
        }

        // TrainingPhrases
        var trainingPhrases3 []*proto3.Intent_TrainingPhrase
        for _, tp2 := range intent2.TrainingPhrases {
                if tp2.Type == proto2.Intent_TrainingPhrase_TEMPLATE {
                        logIssue("Intent<"+intent2.DisplayName+">.TrainingPhrases",
                                "This intent has a training phrase that uses a template (@...) training phrase type. CX does not support this.")
                }
                var parts3 []*proto3.Intent_TrainingPhrase_Part
                for _, part2 := range tp2.Parts {
                        parts3 = append(parts3, &proto3.Intent_TrainingPhrase_Part{
                                Text:        part2.Text,
                                ParameterId: part2.Alias,
                        })
                }
                trainingPhrases3 = append(trainingPhrases3, &proto3.Intent_TrainingPhrase{
                        Parts:       parts3,
                        RepeatCount: 1,
                })
        }
        intent3.TrainingPhrases = trainingPhrases3

        // Action
        if len(intent2.Action) > 0 {
                logIssue("Intent<"+intent2.DisplayName+">.Action",
                        "This intent sets the action field. Use a fulfillment webhook tag instead.")
        }

        // OutputContexts
        if len(intent2.OutputContexts) > 0 {
                logIssue("Intent<"+intent2.DisplayName+">.OutputContexts",
                        "This intent uses context. See the migration guide for alternatives.")
        }

        // ResetContexts
        if intent2.ResetContexts {
                logIssue("Intent<"+intent2.DisplayName+">.ResetContexts",
                        "This intent uses context. See the migration guide for alternatives.")
        }

        // Parameters
        var parameters3 []*proto3.Intent_Parameter
        for _, p2 := range intent2.Parameters {
                if len(p2.Value) > 0 && p2.Value != "$"+p2.DisplayName {
                        logIssue("Intent<"+intent2.DisplayName+">.Parameters<"+p2.DisplayName+">.Value",
                                "This field is not set to $parameter-name. This feature is not supported by CX. See: https://cloud.google.com/dialogflow/es/docs/intents-actions-parameters#valfield.")
                }
                if len(p2.DefaultValue) > 0 {
                        logIssue("Intent<"+intent2.DisplayName+">.Parameters<"+p2.DisplayName+">.DefaultValue",
                                "This intent parameter is using a default value. CX intent parameters do not support default values, but CX page form parameters do. This parameter should probably become a form parameter.")
                }
                if p2.Mandatory {
                        logIssue("Intent<"+intent2.DisplayName+">.Parameters<"+p2.DisplayName+">.Mandatory",
                                "This intent parameter is marked as mandatory. CX intent parameters do not support mandatory parameters, but CX page form parameters do. This parameter should probably become a form parameter.")
                }
                for _, prompt := range p2.Prompts {
                        logIssue("Intent<"+intent2.DisplayName+">.Parameters<"+p2.DisplayName+">.Prompts",
                                "This intent parameter has a prompt. Use page form parameter prompts instead. Prompt: "+prompt)
                }
                if len(p2.EntityTypeDisplayName) == 0 {
                        p2.EntityTypeDisplayName = "@sys.any"
                        logIssue("Intent<"+intent2.DisplayName+">.Parameters<"+p2.DisplayName+">.EntityTypeDisplayName",
                                "This intent parameter does not have an entity type. CX requires an entity type for all parameters..")
                }
                parameters3 = append(parameters3, &proto3.Intent_Parameter{
                        Id:         p2.DisplayName,
                        EntityType: convertParameterEntityType(intent2.DisplayName, p2.DisplayName, p2.EntityTypeDisplayName),
                        IsList:     p2.IsList,
                })
                //fmt.Printf("Converted parameter: %+v\n", parameters3[len(parameters3)-1])
        }
        intent3.Parameters = parameters3

        // Messages
        for _, message := range intent2.Messages {
                m, ok := message.Message.(*proto2.Intent_Message_Text_)
                if ok {
                        for _, t := range m.Text.Text {
                                warnings := ""
                                if strings.Contains(t, "#") {
                                        warnings += " This message may contain a context parameter reference, but CX does not support this."
                                }
                                if strings.Contains(t, ".original") {
                                        warnings += " This message may contain a parameter reference suffix of '.original', But CX only supports this for intent parameters (not session parameters)."
                                }
                                if strings.Contains(t, ".recent") {
                                        warnings += " This message may contain a parameter reference suffix of '.recent', but CX does not support this."
                                }
                                if strings.Contains(t, ".partial") {
                                        warnings += " This message may contain a parameter reference suffix of '.partial', but CX does not support this."
                                }
                                logIssue("Intent<"+intent2.DisplayName+">.Messages",
                                        "This intent has a response message. Use fulfillment instead."+warnings+" Message: "+t)
                        }
                } else {
                        logIssue("Intent<"+intent2.DisplayName+">.Messages",
                                "This intent has a non-text response message. See the rich response message information in the migration guide.")
                }
                if message.Platform != proto2.Intent_Message_PLATFORM_UNSPECIFIED {
                        logIssue("Intent<"+intent2.DisplayName+">.Platform",
                                "This intent has a message with a non-default platform. See the migration guide for advice.")
                }
        }

        return intent3
}

// migrateEntities migrates ES entities to your CX agent
func migrateEntities(ctx context.Context) error {
        var err error

        // Create ES client
        var client2 *v2.EntityTypesClient
        options2 := []option.ClientOption{}
        if len(*v2Region) > 0 {
                options2 = append(options2,
                        option.WithEndpoint(*v2Region+"-dialogflow.googleapis.com:443"))
        }
        client2, err = v2.NewEntityTypesClient(ctx, options2...)
        if err != nil {
                return err
        }
        defer client2.Close()
        var parent2 string
        if len(*v2Region) == 0 {
                parent2 = fmt.Sprintf("projects/%s/agent", *v2Project)
        } else {
                parent2 = fmt.Sprintf("projects/%s/locations/%s/agent", *v2Project, *v2Region)
        }

        // Create CX client
        var client3 *v3.EntityTypesClient
        options3 := []option.ClientOption{}
        if len(*v3Region) > 0 {
                options3 = append(options3,
                        option.WithEndpoint(*v3Region+"-dialogflow.googleapis.com:443"))
        }
        client3, err = v3.NewEntityTypesClient(ctx, options3...)
        if err != nil {
                return err
        }
        defer client3.Close()
        parent3 := fmt.Sprintf("projects/%s/locations/%s/agents/%s", *v3Project, *v3Region, *v3Agent)

        // Read each V2 entity type, convert, and write to V3
        request2 := &proto2.ListEntityTypesRequest{
                Parent: parent2,
        }
        it2 := client2.ListEntityTypes(ctx, request2)
        for {
                var et2 *proto2.EntityType
                et2, err = it2.Next()
                if err == iterator.Done {
                        break
                }
                if err != nil {
                        return err
                }
                fmt.Printf("Entity Type: %s\n", et2.DisplayName)

                if *dryRun {
                        convertEntityType(et2)
                        continue
                }

                request3 := &proto3.CreateEntityTypeRequest{
                        Parent:     parent3,
                        EntityType: convertEntityType(et2),
                }
                et3, err := client3.CreateEntityType(ctx, request3)
                entityTypeShortToLong[et3.DisplayName] = et3.Name
                if err != nil {
                        return err
                }

                // ES and CX each have a quota limit of 60 design-time requests per minute
                time.Sleep(2 * time.Second)
        }
        return nil
}

// migrateIntents migrates intents to your CX agent
func migrateIntents(ctx context.Context) error {
        var err error

        // Create ES client
        var client2 *v2.IntentsClient
        options2 := []option.ClientOption{}
        if len(*v2Region) > 0 {
                options2 = append(options2,
                        option.WithEndpoint(*v2Region+"-dialogflow.googleapis.com:443"))
        }
        client2, err = v2.NewIntentsClient(ctx, options2...)
        if err != nil {
                return err
        }
        defer client2.Close()
        var parent2 string
        if len(*v2Region) == 0 {
                parent2 = fmt.Sprintf("projects/%s/agent", *v2Project)
        } else {
                parent2 = fmt.Sprintf("projects/%s/locations/%s/agent", *v2Project, *v2Region)
        }

        // Create CX client
        var client3 *v3.IntentsClient
        options3 := []option.ClientOption{}
        if len(*v3Region) > 0 {
                options3 = append(options3,
                        option.WithEndpoint(*v3Region+"-dialogflow.googleapis.com:443"))
        }
        client3, err = v3.NewIntentsClient(ctx, options3...)
        if err != nil {
                return err
        }
        defer client3.Close()
        parent3 := fmt.Sprintf("projects/%s/locations/%s/agents/%s", *v3Project, *v3Region, *v3Agent)

        // Read each V2 entity type, convert, and write to V3
        request2 := &proto2.ListIntentsRequest{
                Parent:     parent2,
                IntentView: proto2.IntentView_INTENT_VIEW_FULL,
        }
        it2 := client2.ListIntents(ctx, request2)
        for {
                var intent2 *proto2.Intent
                intent2, err = it2.Next()
                if err == iterator.Done {
                        break
                }
                if err != nil {
                        return err
                }
                fmt.Printf("Intent: %s\n", intent2.DisplayName)
                intent3 := convertIntent(intent2)
                if intent3 == nil {
                        continue
                }

                if *dryRun {
                        continue
                }

                request3 := &proto3.CreateIntentRequest{
                        Parent: parent3,
                        Intent: intent3,
                }
                _, err := client3.CreateIntent(ctx, request3)
                if err != nil {
                        return err
                }

                // ES and CX each have a quota limit of 60 design-time requests per minute
                time.Sleep(2 * time.Second)
        }
        return nil
}

// checkFlags checks commandline flags
func checkFlags() error {
        flag.Parse()
        if len(*v2Project) == 0 {
                return fmt.Errorf("Need to supply es-project-id flag")
        }
        if len(*v3Project) == 0 {
                return fmt.Errorf("Need to supply cx-project-id flag")
        }
        if len(*v2Region) == 0 {
                fmt.Printf("No region supplied for ES, using default\n")
        }
        if len(*v3Region) == 0 {
                return fmt.Errorf("Need to supply cx-region-id flag")
        }
        if len(*v3Agent) == 0 {
                return fmt.Errorf("Need to supply cx-agent-id flag")
        }
        if len(*outFile) == 0 {
                return fmt.Errorf("Need to supply out-file flag")
        }
        return nil
}

// closeFile is used as a convenience for defer
func closeFile(f *os.File) {
        err := f.Close()
        if err != nil {
                fmt.Fprintf(os.Stderr, "ERROR closing CSV file: %v\n", err)
                os.Exit(1)
        }
}

func main() {
        if err := checkFlags(); err != nil {
                fmt.Fprintf(os.Stderr, "ERROR checking flags: %v\n", err)
                os.Exit(1)
        }
        ctx := context.Background()
        if err := migrateEntities(ctx); err != nil {
                fmt.Fprintf(os.Stderr, "ERROR migrating entities: %v\n", err)
                os.Exit(1)
        }
        if err := migrateIntents(ctx); err != nil {
                fmt.Fprintf(os.Stderr, "ERROR migrating intents: %v\n", err)
                os.Exit(1)
        }
        csvFile, err := os.Create(*outFile)
        if err != nil {
                fmt.Fprintf(os.Stderr, "ERROR opening output file: %v", err)
                os.Exit(1)
        }
        defer closeFile(csvFile)
        csvWriter := csv.NewWriter(csvFile)
        if err := csvWriter.WriteAll(issues); err != nil {
                fmt.Fprintf(os.Stderr, "ERROR writing CSV output file: %v", err)
                os.Exit(1)
        }
        csvWriter.Flush()
}

