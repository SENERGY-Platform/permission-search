package worker

import (
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
)

type Query interface {
	GetClient() *elastic.Client
	GetResourceEntry(kind string, resource string) (result model.Entry, version model.ResourceVersion, err error)
	ResourceExists(kind string, resource string) (exists bool, err error)
}
