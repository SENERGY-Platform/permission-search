<a href="https://github.com/SENERGY-Platform/permission-search/actions/workflows/tests.yml" rel="nofollow">
    <img src="https://github.com/SENERGY-Platform/permission-search/actions/workflows/tests.yml/badge.svg?branch=master" alt="Tests" />
</a>


Service to search for resources with added permissions. 
Receives resources from kafka and saves it to elastic search. 
Resources will be enriched with permission information. 
A HTTP-API provides endpoints to request for resources where the requesting user has selected permissions.

## Modes 
The Project can be started as:
- Worker: only handle change commands received from Kafka-Events, without starting the HTTP-API. `./permission-search -mode=worker`
- Query: only start the HTTP-API without starting a Kafka-Consumer. `./permission-search -mode=query` as 
- Standalone: start HTTP-API and Kafka-Consumer. Default behavior with `./permission-search` or `./permission-search -mode=standalone`

## Events
Data-input to the elasticsearch database id done by kafka events.

### Permission-Events
The change of permissions is governed by the permissions service. This is done by event-messaging.
A event-message is a json-object with `command`, `Kind`, `Resource`, `User`, `Group` and `Right` as possible fields.
* `command`: decides if `"PUT"` or `"DELETE"`
* `Kind`: for which resource-kind (for example device or process) should the permissions be changed.
* `Resource`: for which specific resource id should the permissions be changed.
* `User`: For which user id should the permission be changed. mutual exclusive with the `Group` field.
* `Group`: For which group name should the permission be changed. mutual exclusive with the `User` field.
* `Right`: Which right should be set. Only evaluated if `command` is equal to `"PUT"`. Is a string with letters representing rights (for example `"rwx"`):
    * `r`: read right.
    * `w`: wright right.
    * `x`: execute right.
    * `a`: administration right.

The following events are possible:
* Set-Group-Permission-Message: where `command` equals `"PUT"` and the `Group` field is not empty. Expects `Kind`, `Resource` and `Right` to be set.
* Set-User-Permission-Message: where `command` equals `"PUT"` and the `User` field is not empty. Expects `Kind`, `Resource` and `Right` to be set.
* Remove-Group-Permission-Message: where `command` equals `"DELETE"` and the `Group` field is not empty. Expects `Kind` and `Resource` to be set.
* Remove-User-Permission-Message: where `command` equals `"DELETE"` and the `User` field is not empty. Expects `Kind` and `Resource` to be set.


### Resource-Events
changes to resource-features are handled by resource-events. A resource-kind is equal to the topic of the event-messages. 
The following fields are expected: 
* `command`: PUT or DELETE.
* `id`: the resource-id
* `owner`: the creator/owner of the resource. will only evaluated if the resource is not existing prior to this event.

other fields are allowed and will be evaluated according to the resource-config

## HTTP-API V2
### GET /v2/:resource
Lists resources with a similar response as `/jwt/search/:resource_kind/:query/:right`.
Query-Parameter may be:
- limit: `/v2/aspects?limit=20`
- offset: `/v2/aspects?offset=40`
- rights: `/v2/aspects?rights=rw`, default 'r', filters by needed rights
- sort: `/v2/aspects?sort=name.desc`
- search: `/v2/aspects?search=someText`, may not be used in combination with the 'filter' or 'ids' query-parameter 
- filter: `/v2/aspects?filter=name:aspect4_name`, may not be used in combination with the 'search' or 'ids' query-parameter 
- ids: `/v2/aspects?ids=aspect3,aspect2,aspect1`, may not be used in combination with the 'search' or 'filter' query-parameter 

### HEAD /v2/:resource/:id
similar to GET `/jwt/check/:resource_kind/:resource_id/:right`, where the right is passed as query-parameter 'rights'

### GET /v2/:resource/:id/access
similar to GET `/jwt/check/:resource_kind/:resource_id/:right/bool`, where the right is passed as query-parameter 'rights'

### POST /v2/query
reference apiv2_test.go fo details.

## HTTP-API V1

* GET `/administrate/exists/:resource_kind/:resource`: checks if resource exists. returns boolean json.
* GET `/administrate/rights/:resource_kind`: returns a json with the resources the requesting user has admin rights for. With all user and group rights listed.
* GET `/administrate/rights/:resource_kind/get/:resource`: returns a json with the resource if requesting user has admin rights for. With all user and group rights listed.
* GET `/administrate/rights/:resource_kind/query/:query/:limit/:offset`: returns a json with the resources where the requesting user has admin rights and the resource is searchable by the query. With all user and group rights listed.
* GET `/jwt/search/:resource_kind/:query/:right`: searches for resources the requesting user has matching rights
* GET `/jwt/select/:resource_kind/:field/:value/:right`: searches for resources where the field equals the value and the user has matching rights
* GET `/jwt/list/:resource_kind/:right`: list the resources where the requesting user has matching rights
* GET `/jwt/check/:resource_kind/:resource_id/:right`: checks if requesting user has matching rights to resource. returns code 200 with json `{"status": "ok"}` if yes and code 401 if not.
* GET `/jwt/check/:resource_kind/:resource_id/:right/bool`: checks if requesting user has matching rights to resource. returns true if yes and false if not.
* POST `/ids/check/:resource_kind/:right`: like `/jwt/check/:resource_kind/:resource_id/:right/bool` in bulk where the ids for resource_id are transmitted as a list in the request body.
* POST `/ids/select/:resource_kind/:right`: returns resources where the id is in the id-list from the request-body and the requesting user has matching rights.
* GET `/export`: exports the whole database to json.
* PUT `/import`: imports the result of a export.
* POST `/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature/:direction`: like `/jwt/search/:resource_kind/:query/:right` but with additional user-defined selection-filters.
* POST `/jwt/list/:resource_kind/:right/:limit/:offset/:orderfeature/:direction`: like `/jwt/list/:resource_kind/:right` but with additional user-defined selection-filters.


### Postfix-Routes
These routes can be appended on most routes to define sorting and paging.

* `/:limit/:offset` 
    * returns maximal `limit` results.
    * skips `offset` documents.
* `/:limit/:offset/:order_by/asc` 
    * returns maximal `limit` results.
    * skips `offset` documents.
    * orders by field `order_by` ascending.
    * `order_by` may have `field.subfield` syntax.
    * `order_by` must be descibed in ElasticMapping.
* `/:limit/:offset/:order_by/desc` 
    * returns maximal `limit` results
    * skips `offset` documents
    * orders by field `order_by` descending.
    * `order_by` may have `field.subfield` syntax.
    * `order_by` must be descibed in ElasticMapping.

### User-Defined-Selection


#### Selection-Or
Used to combine a list of other Selections/Conditions. At least one condition must apply.

**Example:**
```
{
    "or": [
        {"condition": {"feature": "write.user", "operation": "==", "ref": "jwt.user"}},
        {"condition": {"feature": "write.group", "operation": "any_value_in_feature", "ref": "jwt.groups"}}
    ]
}
```

#### Selection-And
Used to combine a list of other Selections/Conditions. All conditions must apply.

**Example:**
```
{
    "and": [
        {
            "or": [
                {"condition": {"feature": "read.user", "operation": "==", "ref": "jwt.user"}},
                {"condition": {"feature": "read.group", "operation": "any_value_in_feature", "ref": "jwt.groups"}}
            ]
        },
        {
            "or": [
                {"condition": {"feature": "write.user", "operation": "==", "ref": "jwt.user"}},
                {"condition": {"feature": "write.group", "operation": "any_value_in_feature", "ref": "jwt.groups"}}
            ]
        }
    ]
}
```

#### Selection-Condition
Adds a Filter/Condition to the elasticsearch query. A `condition` has the following fields:

* `feature`: (string) reference to a feature saved in the elasticsearch document. May contain `'.'` to traverse (for example `device.name`)
* `operation`: (string) operation that will be executed to determine the result of the condition.
* `value`: (anything) value on which the operation can be executed. The type of the value is determined by the operation and target_feature.
* `ref`: (string) uses predefined references as value.

Currently valid operations are:

* `==`:
    * checks equality with the feature.
    * Uses the `value` field if set, if not it tries to use the `ref` reference.
    * If neither is set the condition will check the non-existence of the `feature`. For example `{"feature": "kind", "operation":"==", "value":null}` searches for documents where the field `kind` does not exist.
    * `{"feature": "name", "operation":"==", "value":"foo"}` searches for documents where the field `name` is equal to `"foo"`.
* `!=`:
    * not `==`.
    * `{"feature": "kind", "operation":"!=", "value":null}` searches for documents where the field `kind` does exist.
    * `{"feature": "name", "operation":"!=", "value":"foo"}` searches for documents where the field `name` is equal to `"foo"`.
* `any_value_in_feature`:
    * interprets the `value` or `ref` as list 
    * checks if any of the list-entries matches the `feature`.
    * the `feature` may be a list but can also be a single value.
        * if list: any target matches any value
        * if single element: any value matches target

Currently valid `ref` values are:

* `"jwt.user"`: (string) uses the user-id that was transmitted by the JWT-Authorisation-Token in the HTTP-Request.
* `"jwt.groups"`: ([]string) uses the groups that where transmitted by the JWT-Authorisation-Token in the HTTP-Request.


## Resource-Config
The Config-Field `Resources` is a map from event topics to a resource-configuration. This configuration consists of the fields `Features` and `InitialGroupRights`.

### Features
Describes how the event should be transformed to a new map (`map[string]interface{}`).
Features consists of a list of descriptions, where each entry describes one field. These descriptions contain the following fields:
* `name`: (string) name of the feature
* `path`: (string) json-path, used on the event to get the value of the field (https://github.com/JumboInteractiveLimited/jsonpath)

### InitialGroupRights
This field describes which groups with which rights a resource initially should get. It is a Map form group-name to rights string.

### Example    
```
{
    ...
    "processmodel":{
        "Features":[
            {"Name": "name", "Path": "$.processmodel.process.definitions.process._id+"},
            {"Name": "date", "Path": "$.processmodel.date+"},
            {"Name": "svg", "Path": "$.processmodel.svg+"},
            {"Name": "publish", "Path": "$.processmodel.publish+"},
            {"Name": "parent_id", "Path": "$.processmodel.parent_id+"}
        ],
        "InitialGroupRights":{"admin": "rwxa"}
    },
    ...
}
```

## ElasticMapping

This section will be used for the Mapping in elasticsearch https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping.html.
The configuration for each resource will be placed under `mapping.doc.properties`.
permissionsearch prepares a index for searches, if you want a field to be searchable in the http-api use `"copy_to": "feature_search"`.
Types other then "Keyword" may influence results of `where` and `queries` by running elasticsearch analysis on this field (for example stemming).

This Section will only be used if the configuration-field `CreateIndex` equal to `"true"` is.
If this is not the case no index will be created and an error will be thrown if no index exists.
Automatic creation of indexes with ElasticMapping is only with small or prototypical applications useful. Or if the mapping is static and will never change.
If you need more control over a ES-Cluster please read the chapter Mapping-Update-On-ES and create/update your indexes manually.

**Example:**

```
"elastic_mapping": {
    "simple_resource": {
      "name":    {"type": "keyword"},
      "devices": {"type": "keyword"},
      "id":      {"type": "keyword"},
      "command": {"type": "keyword"}
    },
    "complex_resource":{
      "device":{
        "properties": {
          "name":         {"type": "keyword", "copy_to": "feature_search"},
          "description":  {"type": "text",    "copy_to": "feature_search"},
          "usertag":      {"type": "keyword", "copy_to": "feature_search"},
          "tag":          {"type": "keyword", "copy_to": "feature_search"},
          "devicetype":   {"type": "keyword"},
          "uri":          {"type": "keyword"},
          "img":          {"type": "keyword"}
        }
      },
      "gw":{
        "properties": {
          "name":         {"type": "keyword", "copy_to": "feature_search"}
        }
      }
    }
  }
```

## Mapping-Update

**With Compiled Executable:**
```
./permission-search update-indexes
```

**Update Selected Indexes:**
```
./permission-search update-indexes device-types devices characteristics
```

**In Docker Container:**
```
./app update-indexes
```

## Mapping-Update Manually
### Add-Field
```
PUT https://api.sepl.infai.org/permission/search-db/processmodel/_mapping/resource
{  
 "properties":{  
    "admin_groups":{  
       "type":"keyword"
    },
    "admin_users":{  
       "type":"keyword"
    },
    "creator":{  
       "type":"keyword"
    },
    "execute_groups":{  
       "type":"keyword"
    },
    "execute_users":{  
       "type":"keyword"
    },
    "feature_search":{  
       "analyzer":"autocomplete",
       "search_analyzer":"standard",
       "type":"text"
    },
    "features":{  
       "properties":{  
          "date":{  
             "type":"date"
          },
          "name":{  
             "copy_to":"feature_search",
             "type":"keyword"
          },
          "parent_id":{  
             "type":"keyword"
          },
          "publish":{  
             "type":"boolean"
          },
          "description":{  
             "type":"keyword"
          }
       }
    },
    "read_groups":{  
       "type":"keyword"
    },
    "read_users":{  
       "type":"keyword"
    },
    "resource":{  
       "type":"keyword"
    },
    "write_groups":{  
       "type":"keyword"
    },
    "write_users":{  
       "type":"keyword"
    }
 }
}
```

### Reindexing:

**1. new index:**
```
PUT https://api.sepl.infai.org/permission/search-db/gateway_v2
{  
   "mappings":{  
      "resource":{  
         "properties":{  
            "admin_groups":{  
               "type":"keyword"
            },
            "admin_users":{  
               "type":"keyword"
            },
            "creator":{  
               "type":"keyword"
            },
            "execute_groups":{  
               "type":"keyword"
            },
            "execute_users":{  
               "type":"keyword"
            },
            "feature_search":{  
               "analyzer":"autocomplete",
               "search_analyzer":"standard",
               "type":"text"
            },
            "features":{"properties":{"devices":{"type":"keyword"},"name":{"copy_to":"feature_search","type":"keyword"}}},
            "read_groups":{  
               "type":"keyword"
            },
            "read_users":{  
               "type":"keyword"
            },
            "resource":{  
               "type":"keyword"
            },
            "write_groups":{  
               "type":"keyword"
            },
            "write_users":{  
               "type":"keyword"
            }
         }
      }
   },
   "settings":{  
     "analysis":{  
     "analyzer":{  
        "autocomplete":{  
           "filter":[  
              "lowercase",
              "autocomplete_filter"
           ],
           "tokenizer":"standard",
           "type":"custom"
        }
     },
     "filter":{  
        "autocomplete_filter":{  
           "max_gram":20,
           "min_gram":1,
           "type":"edge_ngram"
        }
     }
      }
   }
}
```

**2. reindexing:** https://www.elastic.co/guide/en/elasticsearch/reference/current/docs-reindex.html
```
POST _reindex
{
  "source": {
    "index": "gateway_v1"
  },
  "dest": {
    "index": "gateway_v2"
  }
}
```
```
POST _reindex
{
  "source": {
  "remote": {
      "host": "http://elastic.permissions.rancher.internal"
    },
    "index": "gateway"
  },
  "dest": {
    "index": "gateway_v2"
  }
}
```

**3. alias neu setzen:** https://www.elastic.co/guide/en/elasticsearch/reference/6.2/indices-aliases.html
```
POST /_aliases
{
    "actions" : [
        { "remove" : { "index" : "gateway_v1", "alias" : "gateway" } },
        { "add" : { "index" : "gateway_v2", "alias" : "gateway" } }
    ]
}
```

# Replay Permissions

```
# dry-run for everything
./permission-search replay-permissions

# run for everything
./permission-search replay-permissions do

# dry-run for devices
./permission-search replay-permissions devices

# run for devices and device-groups
./permission-search replay-permissions do devices device-groups
```