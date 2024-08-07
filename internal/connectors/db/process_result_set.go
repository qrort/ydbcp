package db

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/result/named"
	"ydbcp/internal/types"
)

type StructFromResultSet[T any] func(result result.Result) (*T, error)

type InterfaceFromResultSet[T any] func(result result.Result) (T, error)

func StringOrDefault(str *string, def string) string {
	if str == nil {
		return def
	}
	return *str
}

func StringOrEmpty(str *string) string {
	return StringOrDefault(str, "")
}

func ReadBackupFromResultSet(res result.Result) (*types.Backup, error) {
	var (
		backupId     [16]byte
		containerId  string
		databaseName string
		s3endpoint   *string
		s3region     *string
		s3pathprefix *string
		status       *string
	)

	err := res.ScanNamed(
		named.Required("id", &backupId),
		named.Required("container_id", &containerId),
		named.Required("database", &databaseName),
		named.Optional("s3_endpoint", &s3endpoint),
		named.Optional("s3_region", &s3region),
		named.Optional("s3_path_prefix", &s3pathprefix),
		named.Optional("status", &status),
	)
	if err != nil {
		return nil, err
	}

	id, err := uuid.FromBytes(backupId[:])

	if err != nil {
		return nil, err
	}

	return &types.Backup{
		ID:           types.ObjectID(id),
		ContainerID:  containerId,
		DatabaseName: databaseName,
		S3Endpoint:   StringOrEmpty(s3endpoint),
		S3Region:     StringOrEmpty(s3region),
		S3Bucket:     "",
		S3PathPrefix: StringOrEmpty(s3pathprefix),
		Status:       StringOrDefault(status, types.BackupStateUnknown),
	}, nil
}

func ReadOperationFromResultSet(res result.Result) (types.Operation, error) {
	var (
		operationId   types.ObjectID
		containerId   string
		operationType string

		operationStateBuf *string
		backupId          *types.ObjectID
		ydbOperationId    *string
		database          *string
	)
	err := res.ScanNamed(
		named.Required("id", &operationId),
		named.Required("container_id", &containerId),
		named.Required("type", &operationType),

		named.Optional("status", &operationStateBuf),
		named.Optional("backup_id", &backupId),
		named.Optional("operation_id", &ydbOperationId),
		named.Optional("database", &database),
	)
	if err != nil {
		return nil, err
	}
	operationState := types.OperationStateUnknown
	if operationStateBuf != nil {
		operationState = types.OperationState(*operationStateBuf)
	}
	if operationType == string(types.OperationTypeTB) {
		if backupId == nil || database == nil || ydbOperationId == nil {
			return nil, fmt.Errorf("failed to read required fields of operation %s", operationId.String())
		}
		return &types.TakeBackupOperation{
			Id:                  operationId,
			BackupId:            types.ObjectID(*backupId),
			ContainerID:         containerId,
			State:               operationState,
			Message:             "",
			YdbConnectionParams: types.GetYdbConnectionParams(*database),
			YdbOperationId:      *ydbOperationId,
		}, nil
	} else if operationType == string(types.OperationTypeRB) {
		if backupId == nil || database == nil || ydbOperationId == nil {
			return nil, fmt.Errorf("failed to read required fields of operation %s", operationId.String())
		}
		return &types.RestoreBackupOperation{
			Id:                  operationId,
			BackupId:            types.ObjectID(*backupId),
			ContainerID:         containerId,
			State:               operationState,
			Message:             "",
			YdbConnectionParams: types.GetYdbConnectionParams(*database),
			YdbOperationId:      *ydbOperationId,
		}, nil
	}

	return &types.GenericOperation{Id: operationId}, nil
}
