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

package lib

import (
	"context"

	"log"

	elastic "github.com/olivere/elastic/v7"
)

func SetUserRight(kind string, resource string, user string, rights string) (err error) {
	ctx := context.Background()
	entry, version, err := getResourceEntry(ctx, kind, resource)
	if err != nil {
		return err
	}
	entry.removeUserRights(user)
	entry.addUserRights(user, rights)
	_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(resource).Version(version).BodyJson(entry).Do(ctx)
	return
}

func SetGroupRight(kind string, resource string, group string, rights string) (err error) {
	ctx := context.Background()
	entry, version, err := getResourceEntry(ctx, kind, resource)
	if err != nil {
		return err
	}
	entry.removeGroupRights(group)
	entry.addGroupRights(group, rights)
	_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(resource).Version(version).BodyJson(entry).Do(ctx)
	return
}

func DeleteUserRight(kind string, resource string, user string) (err error) {
	ctx := context.Background()
	entry, version, err := getResourceEntry(ctx, kind, resource)
	if err != nil {
		return err
	}
	entry.removeUserRights(user)
	_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(resource).Version(version).BodyJson(entry).Do(ctx)
	return
}

func DeleteGroupRight(kind string, resource string, group string) (err error) {
	ctx := context.Background()
	entry, version, err := getResourceEntry(ctx, kind, resource)
	if err != nil {
		return err
	}
	entry.removeGroupRights(group)
	_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(resource).Version(version).BodyJson(entry).Do(ctx)
	return
}

func UpdateFeatures(kind string, msg []byte, command CommandWrapper) (err error) {
	features, err := MsgToFeatures(kind, msg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	exists, err := resourceExists(ctx, kind, command.Id)
	if err != nil {
		return err
	}
	if exists {
		entry, version, err := getResourceEntry(ctx, kind, command.Id)
		if err != nil {
			return err
		}
		entry.Features = features
		if entry.Creator == "" && len(entry.AdminUsers) > 0 {
			entry.Creator = entry.AdminUsers[0]
		}
		_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(command.Id).Version(version).BodyJson(entry).Do(ctx)
	} else {
		entry := Entry{Resource: command.Id, Features: features, Creator: command.Owner}
		entry.setDefaultPermissions(kind, command.Owner)
		_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(command.Id).BodyJson(entry).Do(ctx)
	}
	return

}

func DeleteFeatures(kind string, command CommandWrapper) (err error) {
	ctx := context.Background()
	exists, err := GetClient().Exists().Index(kind).Type(ElasticPermissionType).Id(command.Id).Do(ctx)
	if err != nil {
		log.Println("ERROR: DeleteFeatures() check existence ", err)
		return err
	}
	if exists {
		_, err = GetClient().Delete().Index(kind).Type(ElasticPermissionType).Id(command.Id).Do(ctx)
	}
	return
}

func DeleteUser(user string) (err error) {
	for kind := range Config.Resources {
		err = DeleteUserFromResourceKind(kind, user)
		if err != nil {
			return
		}
	}
	return
}

func DeleteUserFromResourceKind(kind string, user string) (err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Should(
		elastic.NewTermQuery("admin_users", user),
		elastic.NewTermQuery("read_users", user),
		elastic.NewTermQuery("write_users", user),
		elastic.NewTermQuery("execute_users", user))
	result, err := GetClient().Search().Index(kind).Type(ElasticPermissionType).Version(true).Query(query).Do(ctx)
	if err != nil {
		return err
	}
	for _, hit := range result.Hits.Hits {
		if hit.Type != ElasticPermissionType {
			log.Println("DEBUG: DeleteUserFromResourceKind: unknown type", hit.Type)
			continue
		}
		err = DeleteUserRight(kind, hit.Id, user)
		if err != nil {
			return err
		}
		//TODO: delete resource if last admin??
	}
	return
}
