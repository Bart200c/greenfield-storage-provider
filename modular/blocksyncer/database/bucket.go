package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/forbole/juno/v4/common"
	"github.com/forbole/juno/v4/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/bnb-chain/greenfield-storage-provider/pkg/log"
	"github.com/bnb-chain/greenfield-storage-provider/store/bsdb"
)

func (db *DB) SaveBucket(ctx context.Context, bucket *models.Bucket) error {
	return nil
}

func (db *DB) UpdateBucket(ctx context.Context, bucket *models.Bucket) error {
	return nil
}

func (db *DB) SaveBucketToSQL(ctx context.Context, bucket *models.Bucket) (string, []interface{}) {
	stat := db.Db.Session(&gorm.Session{DryRun: true}).Table((&models.Bucket{}).TableName()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bucket_id"}},
		UpdateAll: true,
	}).Create(bucket).Statement

	return stat.SQL.String(), stat.Vars
}

func (db *DB) UpdateBucketToSQL(ctx context.Context, bucket *models.Bucket) (string, []interface{}) {
	stat := db.Db.Session(&gorm.Session{DryRun: true}).Table((&models.Bucket{}).TableName()).Where("bucket_id = ?", bucket.BucketID).Updates(bucket).Statement
	return stat.SQL.String(), stat.Vars
}

func (db *DB) BatchUpdateBucketSize(ctx context.Context, buckets []*models.Bucket) error {
	return db.Db.Table((&models.Bucket{}).TableName()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bucket_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"storage_size", "charge_size"}),
	}).Create(buckets).Error
}

func (db *DB) GetDataMigrationRecord(ctx context.Context, processKey string) (*bsdb.DataMigrationRecord, error) {
	var (
		dataRecord *bsdb.DataMigrationRecord
		err        error
	)
	err = db.Db.Take(&dataRecord, "process_key = ?", processKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return dataRecord, err
}

func (db *DB) UpdateDataMigrationRecord(ctx context.Context, processKey string, isCompleted bool) error {
	return db.Db.Table((&bsdb.DataMigrationRecord{}).TableName()).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "process_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_completed"}),
	}).Create(&bsdb.DataMigrationRecord{ProcessKey: processKey, IsCompleted: isCompleted}).Error
}

func (db *DB) UpdateStorageSizeToSQL(ctx context.Context, objectID common.Hash, bucketName, operation string) (string, []interface{}) {
	tableName := bsdb.GetObjectsTableName(bucketName)
	sql := `UPDATE buckets SET storage_size = storage_size %s CONVERT((SELECT payload_size FROM %s WHERE object_id = ?), DECIMAL(65,0)) WHERE bucket_name = ?`
	vars := []interface{}{objectID, bucketName}
	finalSQL := fmt.Sprintf(sql, operation, tableName)
	return finalSQL, vars
}

func (db *DB) UpdateChargeSizeToSQL(ctx context.Context, objectID common.Hash, bucketName, operation string) (string, []interface{}) {
	tableName := bsdb.GetObjectsTableName(bucketName)
	sql := `UPDATE buckets SET charge_size = charge_size %s CASE WHEN (CAST((SELECT payload_size FROM %s WHERE object_id = ?)AS DECIMAL(65,0)) < 128000) THEN CAST(128000 AS DECIMAL(65,0)) ELSE CAST((SELECT payload_size FROM %s WHERE object_id = ?) AS DECIMAL(65,0)) END WHERE bucket_name = ?`
	vars := []interface{}{objectID, objectID, bucketName}
	finalSql := fmt.Sprintf(sql, operation, tableName, tableName)
	log.Infof("AddChargeSizeToSQL sql:%s", finalSql)
	return finalSql, vars
}
