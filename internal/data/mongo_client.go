package data

import (
	"Hamburger/internal/config"
	"Hamburger/internal/data/model"
	"Hamburger/internal/logger"
	"time"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mongo客户端

func InitMongo() {
	cf := config.Get()
	logger.L().Info().Msg("init mongodb")
	err := mgm.SetDefaultConfig(&mgm.Config{CtxTimeout: 5 * time.Second}, cf.Database.Mongo.Database, options.Client().ApplyURI(cf.Database.Mongo.URL))
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("failed to connect to mongo")
		return
	}
}

// GetAppFromMongo 获取域名映射表
func GetAppFromMongo() []model.DaoAPP {
	var data []model.DaoAPP
	err := mgm.Coll(&model.DaoAPP{}).SimpleFind(&data, bson.M{})
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("get data from mongo failed")
		return nil
	}
	return data
}
