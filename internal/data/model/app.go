package model

import (
	"github.com/JJApplication/octopus_meta"
	"github.com/kamva/mgm/v3"
)

type App struct {
	Meta octopus_meta.App
}

type DaoAPP struct {
	mgm.DefaultModel `bson:",inline"`
	App              `bson:"app"`
}

func (a *DaoAPP) CollectionName() string {
	return "microservice"
}
