package master

import (
	"context"
	"encoding/json"
	"github.com/k-si/crongo/common"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var (
	EtcdConn EtcdConnector
	MongoConn MongoConnector
)

type EtcdConnector struct {
	cfg clientv3.Config
	cli *clientv3.Client
}

func InitEtcdConnector() (err error) {
	cfg := clientv3.Config{
		Endpoints:   Cfg.Endpoints,
		DialTimeout: time.Duration(Cfg.DialTimeOut) * time.Millisecond,
	}
	cli, err := clientv3.New(cfg)
	EtcdConn.cfg = cfg
	EtcdConn.cli = cli
	return
}

func (etcd EtcdConnector) Close() error {
	return EtcdConn.cli.Close()
}

func (etcd EtcdConnector) SaveJob(job *common.Job) (err error) {

	var b []byte

	if b, err = json.Marshal(job); err != nil {
		return
	}
	if _, err = etcd.cli.Put(context.TODO(), common.JobDir+job.Name, string(b)); err != nil {
		return
	}

	return
}

func (etcd EtcdConnector) DeleteJob(jobName string) (err error) {
	if _, err = etcd.cli.Delete(context.TODO(), common.JobDir+jobName); err != nil {
		return
	}
	return
}

func (etcd EtcdConnector) ListJob() (jobs []*common.Job, err error) {

	var getResp *clientv3.GetResponse
	jobs = make([]*common.Job, 0)

	if getResp, err = etcd.cli.Get(context.TODO(), common.JobDir, clientv3.WithPrefix()); err != nil {
		return
	}

	for _, kv := range getResp.Kvs {
		job := &common.Job{}
		if err = json.Unmarshal(kv.Value, job); err != nil {
			return
		}
		jobs = append(jobs, job)
	}

	return
}

func (etcd EtcdConnector) KillJob(jobName string) (err error) {

	var (
		grantResp *clientv3.LeaseGrantResponse
	)

	// 租约自动过期
	if grantResp, err = etcd.cli.Grant(context.TODO(), 1); err != nil {
		return
	}
	if _, err = etcd.cli.Put(context.TODO(), common.KillDir+jobName, "", clientv3.WithLease(grantResp.ID)); err != nil {
		return
	}

	return
}

// mongodb
type MongoConnector struct {
	cli        *mongo.Client
	db         *mongo.Database
	collection *mongo.Collection
}

func InitMongoConnector() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(Cfg.ConnectTimeOut)*time.Millisecond)
	defer cancel()
	MongoConn.cli, err = mongo.Connect(ctx, options.Client().ApplyURI(Cfg.ApplyUri))
	MongoConn.db = MongoConn.cli.Database(Cfg.DBName)
	MongoConn.collection = MongoConn.db.Collection(Cfg.CollectionName)
	return
}

func (conn MongoConnector) Close() (err error) {
	return conn.cli.Disconnect(context.Background())
}