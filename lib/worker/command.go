/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"runtime/debug"

	"log"
)

func (this *Worker) SetUserRight(kind string, resource string, user string, rights string) (err error) {
	ctx := context.Background()
	exists, err := this.query.ResourceExists(kind, resource)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, resource)
		if err == model.ErrNotFound {
			log.Println("WARNING: received rights command for none existing resource", kind, resource)
			return nil
		}
		if err != nil {
			return err
		}
		entry.RemoveUserRights(user)
		entry.AddUserRights(user, rights)
		client := this.query.GetClient()
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(resource),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
		return nil
	} else {
		log.Println("WARNING: received rights command for none existing resource", kind, resource)
	}
	return nil
}

func (this *Worker) SetGroupRight(kind string, resource string, group string, rights string) (err error) {
	ctx := context.Background()
	exists, err := this.query.ResourceExists(kind, resource)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, resource)
		if err == model.ErrNotFound {
			log.Println("WARNING: received rights command for none existing resource", kind, resource)
			return nil
		}
		if err != nil {
			return err
		}
		entry.RemoveGroupRights(group)
		entry.AddGroupRights(group, rights)

		client := this.query.GetClient()
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(resource),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
		return nil
	} else {
		log.Println("WARNING: received rights command for none existing resource", kind, resource)
	}
	return nil
}

func (this *Worker) DeleteUserRight(kind string, resource string, user string) (err error) {
	ctx := context.Background()
	exists, err := this.query.ResourceExists(kind, resource)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, resource)
		if err == model.ErrNotFound {
			log.Println("WARNING: received rights command for none existing resource", kind, resource)
			return nil
		}
		if err != nil {
			return err
		}
		entry.RemoveUserRights(user)
		client := this.query.GetClient()
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(resource),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
		return nil
	} else {
		log.Println("WARNING: received rights command for none existing resource", kind, resource)
	}
	return nil
}

func (this *Worker) DeleteGroupRight(kind string, resource string, group string) (err error) {
	ctx := context.Background()
	exists, err := this.query.ResourceExists(kind, resource)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, resource)
		if err == model.ErrNotFound {
			log.Println("WARNING: received rights command for none existing resource", kind, resource)
			return nil
		}
		if err != nil {
			debug.PrintStack()
			return err
		}
		entry.RemoveGroupRights(group)
		client := this.query.GetClient()
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(resource),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
		return nil
	} else {
		log.Println("WARNING: received rights command for none existing resource", kind, resource)
	}
	return nil
}

func (this *Worker) UpdateFeatures(kind string, msg []byte, command model.CommandWrapper) (err error) {
	ctx := this.getTimeout()
	features, err := this.MsgToFeatures(kind, msg)
	if err != nil {
		return err
	}
	exists, err := this.query.ResourceExists(kind, command.Id)
	if err != nil {
		return err
	}
	client := this.query.GetClient()
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, command.Id)
		if err != nil {
			return err
		}
		if entry.Resource == "" {
			log.Printf("WARNING: ignore UpdateFeatures without id %#v\n", command)
			return nil
		}
		entry.Features = features
		if entry.Creator == "" && len(entry.AdminUsers) > 0 {
			entry.Creator = entry.AdminUsers[0]
		}
		if entry.Creator == "" {
			entry.Creator = command.Owner
		}
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(command.Id),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
			//client.Index.WithRefresh("wait_for"), //to slow, don't use
		)
		if err != nil {
			debug.PrintStack()
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			debug.PrintStack()
			return errors.New(resp.String())
		}
	} else {
		entry := model.Entry{Resource: command.Id, Features: features, Creator: command.Owner}
		entry.SetDefaultPermissions(this.config, kind, command.Owner)
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(command.Id),
			client.Index.WithContext(ctx),
			//client.Index.WithRefresh("wait_for"), //to slow, don't use
		)
		if err != nil {
			debug.PrintStack()
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			debug.PrintStack()
			return errors.New(resp.String())
		}
	}
	return nil
}

func (this *Worker) UpdateRights(kind string, msg []byte, command model.CommandWrapper) (err error) {
	ctx := this.getTimeout()
	rights, err := this.MsgToRights(msg)
	if err != nil {
		return err
	}
	if rights == nil {
		log.Println("WARNING: received rights command without new rights")
		return nil
	}
	exists, err := this.query.ResourceExists(kind, command.Id)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := this.query.GetResourceEntry(kind, command.Id)
		if err != nil {
			return err
		}
		if entry.Resource == "" {
			log.Printf("WARNING: ignore UpdateRights without id %#v\n", command)
			return nil
		}
		entry.AdminUsers = []string{}
		entry.AdminGroups = []string{}
		entry.ReadUsers = []string{}
		entry.ReadGroups = []string{}
		entry.WriteUsers = []string{}
		entry.WriteGroups = []string{}
		entry.ExecuteUsers = []string{}
		entry.ExecuteGroups = []string{}
		entry.SetResourceRights(*rights)

		if entry.Creator == "" && len(entry.AdminUsers) > 0 {
			entry.Creator = entry.AdminUsers[0]
		}
		if entry.Creator == "" {
			entry.Creator = command.Owner
		}
		client := this.query.GetClient()
		resp, err := client.Index(
			kind,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(command.Id),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
	}
	return nil
}

func (this *Worker) DeleteFeatures(kind string, command model.CommandWrapper) (err error) {
	ctx := context.Background()
	exists, err := this.query.ResourceExists(kind, command.Id)
	if err != nil {
		log.Println("ERROR: DeleteFeatures() check existence ", err)
		return err
	}
	if exists {
		client := this.query.GetClient()
		resp, err := client.Delete(
			kind,
			command.Id,
			client.Delete.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
	}
	return
}

func (this *Worker) MsgToRights(msg []byte) (result *model.ResourceRightsBase, err error) {
	command := model.CommandWithRights{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return nil, err
	}
	if command.Command != "RIGHTS" {
		return nil, errors.New("MsgToRights() expects Command=='RIGHTS'")
	}
	return command.Rights, nil
}
