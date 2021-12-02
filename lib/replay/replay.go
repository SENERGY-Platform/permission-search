package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/olivere/elastic/v7"
	"runtime/debug"
)

const DefaultBatchSize = 1000

func ReplayPermissions(config configuration.Config, args []string) {
	dryrun := false
	if len(args) == 0 {
		fmt.Println("Dry-Run; to execute use 'do' as the first argument (./permission-search replay-permissions do)")
		dryrun = true
	} else if args[0] != "do" {
		fmt.Println("Dry-Run; to execute use 'do' as the first argument (./permission-search replay-permissions do)")
		dryrun = true
		args = args[1:]
	}
	topics := config.ResourceList
	if len(args) > 0 {
		topics = args
	}
	client, err := elastic.NewClient(elastic.SetURL(config.ElasticUrl), elastic.SetRetrier(query.NewRetrier(config)))
	if err != nil {
		fmt.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var producer *kafka.Producer
	if !dryrun {
		producer, err = kafka.NewProducer(ctx, config.KafkaUrl, config.PermTopic, false)
	}
	if err != nil {
		fmt.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	for _, topic := range topics {
		ReplayPermissionsOfResourceKind(producer, client, topic, DefaultBatchSize)
	}
}

func ReplayPermissionsOfResourceKind(producer *kafka.Producer, client *elastic.Client, kind string, batchSize int) {
	for command := range GetCommands(client, kind, batchSize) {
		msg, err := json.Marshal(command)
		if err != nil {
			fmt.Println("ERROR:", err)
			debug.PrintStack()
			return
		}
		fmt.Println(string(msg))
		if producer != nil {
			err = producer.Produce(command.Resource, msg)
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}
		}
	}
}

func GetCommands(client *elastic.Client, kind string, batchSize int) (commands chan model.PermCommandMsg) {
	commands = make(chan model.PermCommandMsg)
	entries := GetEntries(client, kind, batchSize)
	go func() {
		defer close(commands)
		for entry := range entries {
			userRight := map[string]string{}
			for _, user := range entry.ReadUsers {
				userRight[user] = userRight[user] + "r"
			}
			for _, user := range entry.WriteUsers {
				userRight[user] = userRight[user] + "w"
			}
			for _, user := range entry.ExecuteUsers {
				userRight[user] = userRight[user] + "x"
			}
			for _, user := range entry.AdminUsers {
				userRight[user] = userRight[user] + "a"
			}
			for user, right := range userRight {
				commands <- model.PermCommandMsg{
					Command:  "PUT",
					Kind:     kind,
					Resource: entry.Resource,
					User:     user,
					Right:    right,
				}
			}
			groupRight := map[string]string{}
			for _, group := range entry.ReadGroups {
				groupRight[group] = groupRight[group] + "r"
			}
			for _, group := range entry.WriteGroups {
				groupRight[group] = groupRight[group] + "w"
			}
			for _, group := range entry.ExecuteGroups {
				groupRight[group] = groupRight[group] + "x"
			}
			for _, group := range entry.AdminGroups {
				groupRight[group] = groupRight[group] + "a"
			}
			for group, right := range groupRight {
				commands <- model.PermCommandMsg{
					Command:  "PUT",
					Kind:     kind,
					Resource: entry.Resource,
					Group:    group,
					Right:    right,
				}
			}

		}
	}()
	return commands
}

func GetEntries(client *elastic.Client, kind string, batchSize int) (entries chan model.Entry) {
	lastId := ""
	entries = make(chan model.Entry)
	go func() {
		defer close(entries)
		for {
			query := client.Search().Size(batchSize).Sort("resource", true)
			if lastId != "" {
				query = query.SearchAfter(lastId)
			}
			resp, err := client.Search().Index(kind).Do(context.Background())
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}
			if len(resp.Hits.Hits) == 0 {
				return
			}
			for _, hit := range resp.Hits.Hits {
				entry := model.Entry{}
				err = json.Unmarshal(hit.Source, &entry)
				if err != nil {
					fmt.Println("ERROR:", err)
					debug.PrintStack()
					return
				}
				entries <- entry
				lastId = entry.Resource
			}
		}
	}()
	return entries
}
