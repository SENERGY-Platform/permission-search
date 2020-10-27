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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
	"runtime/debug"

	"log"
)

func (this *Worker) SetUserRight(kind string, resource string, user string, rights string) (err error) {
	ctx := context.Background()
	entry, version, err := this.query.GetResourceEntry(kind, resource)
	if err != nil {
		return err
	}
	entry.RemoveUserRights(user)
	entry.AddUserRights(user, rights)
	_, err = this.query.GetClient().Index().Index(kind).Id(resource).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
	return
}

func (this *Worker) SetGroupRight(kind string, resource string, group string, rights string) (err error) {
	ctx := context.Background()
	entry, version, err := this.query.GetResourceEntry(kind, resource)
	if err != nil {
		return err
	}
	entry.RemoveGroupRights(group)
	entry.AddGroupRights(group, rights)
	_, err = this.query.GetClient().Index().Index(kind).Id(resource).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
	return
}

func (this *Worker) DeleteUserRight(kind string, resource string, user string) (err error) {
	ctx := context.Background()
	entry, version, err := this.query.GetResourceEntry(kind, resource)
	if err != nil {
		return err
	}
	entry.RemoveUserRights(user)
	_, err = this.query.GetClient().Index().Index(kind).Id(resource).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
	return
}

func (this *Worker) DeleteGroupRight(kind string, resource string, group string) (err error) {
	ctx := context.Background()
	entry, version, err := this.query.GetResourceEntry(kind, resource)
	if err != nil {
		debug.PrintStack()
		return err
	}
	entry.RemoveGroupRights(group)
	_, err = this.query.GetClient().Index().Index(kind).Id(resource).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
	return
}

func (this *Worker) UpdateFeatures(kind string, msg []byte, command model.CommandWrapper) (err error) {
	ctx := context.Background()
	features, err := this.MsgToFeatures(kind, msg)
	if err != nil {
		return err
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
		entry.Features = features
		if entry.Creator == "" && len(entry.AdminUsers) > 0 {
			entry.Creator = entry.AdminUsers[0]
		}
		_, err = this.query.GetClient().Index().Index(kind).Id(command.Id).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
		if err != nil {
			return err
		}
	} else {
		entry := model.Entry{Resource: command.Id, Features: features, Creator: command.Owner}
		entry.SetDefaultPermissions(this.config, kind, command.Owner)
		_, err = this.query.GetClient().Index().Index(kind).Id(command.Id).BodyJson(entry).Do(ctx)
		if err != nil {
			return err
		}
	}
	return nil

}

func (this *Worker) DeleteFeatures(kind string, command model.CommandWrapper) (err error) {
	ctx := context.Background()
	exists, err := this.query.GetClient().Exists().Index(kind).Id(command.Id).Do(ctx)
	if err != nil {
		log.Println("ERROR: DeleteFeatures() check existence ", err)
		return err
	}
	if exists {
		_, err = this.query.GetClient().Delete().Index(kind).Id(command.Id).Do(ctx)
	}
	return
}

func (this *Worker) DeleteUser(user string) (err error) {
	for kind := range this.config.Resources {
		err = this.DeleteUserFromResourceKind(kind, user)
		if err != nil {
			return
		}
	}
	return
}

func (this *Worker) DeleteUserFromResourceKind(kind string, user string) (err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Should(
		elastic.NewTermQuery("admin_users", user),
		elastic.NewTermQuery("read_users", user),
		elastic.NewTermQuery("write_users", user),
		elastic.NewTermQuery("execute_users", user))
	result, err := this.query.GetClient().Search().Index(kind).Version(true).Query(query).Do(ctx)
	if err != nil {
		return err
	}
	for _, hit := range result.Hits.Hits {
		err = this.DeleteUserRight(kind, hit.Id, user)
		if err != nil {
			return err
		}
		//TODO: delete resource if last admin??
	}
	return
}
