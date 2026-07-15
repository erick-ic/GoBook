package mogondb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMonDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	monitor := &event.CommandMonitor{
		Started: func(ctx context.Context, e *event.CommandStartedEvent) {
			fmt.Println("<<<<CommandStartedEvent>>>>", e.Command)
		},
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) {},
		Failed:    func(ctx context.Context, e *event.CommandFailedEvent) {},
	}

	opts := options.Client().ApplyURI("mongodb://root:example@localhost:27017").SetMonitor(monitor)

	client, err := mongo.Connect(opts)
	assert.NoError(t, err)

	//// 可选：使用 Ping 验证连接是否成功
	//err = client.Ping(ctx, nil)
	//assert.NoError(t, err)

	//建库
	mdb := client.Database("gobook")
	//建表
	col := mdb.Collection("articles")

	//1.插入数据
	//res, err := col.InsertOne(ctx, Article{
	//	Id:       2,
	//	Title:    "title2",
	//	Content:  "content2",
	//	AuthorId: 21,
	//})
	//assert.NoError(t, err)
	//fmt.Println("res.InsertedID:", res.InsertedID)

	//var result Article
	//err = col.FindOne(ctx, map[string]interface{}{"title": "title1"}).Decode(&result)
	//assert.NoError(t, err)
	//fmt.Printf("Found: %+v\n", result)

	//2.查找数据
	//查找ID=1
	//方式1:
	//var article Article
	//filter := bson.D{bson.E{Key: "id", Value: 1}}
	//err = col.FindOne(ctx, filter).Decode(&article)
	//assert.NoError(t, err)
	//fmt.Printf("%+v\n", article)

	//方式2:
	//article := Article{}
	//err = col.FindOne(ctx, Article{Id: 1}).Decode(&article)
	//if errors.Is(err, mongo.ErrNoDocuments) {
	//	fmt.Println("没有找到数据！")
	//}
	//assert.NoError(t, err)
	//fmt.Printf("%+v\n", article)

	//3.更新数据
	//方式1:
	//filter := bson.D{bson.E{Key: "id", Value: 1}}
	//sets := bson.D{
	//	{Key: "$set", Value: bson.D{
	//		{Key: "title", Value: "title777"},
	//	}},
	//}
	//updateRes, err := col.UpdateMany(ctx, filter, sets)
	//assert.NoError(t, err)
	//fmt.Println("affected:", updateRes.ModifiedCount)

	//方式2:（推荐）
	//filter := bson.D{bson.E{Key: "id", Value: 1}}
	//sets := bson.D{bson.E{Key: "$set", Value: Article{
	//	Title:    "title999",
	//	AuthorId: 999,
	//}}}
	//updateRes, err := col.UpdateMany(ctx, filter, sets)
	//assert.NoError(t, err)
	//fmt.Println("affected:", updateRes.ModifiedCount)

	//4.删除数据
	filter := bson.D{bson.E{Key: "id", Value: 2}}
	deleteRes, err := col.DeleteMany(ctx, filter)
	assert.NoError(t, err)
	fmt.Println("deleted:", deleteRes.DeletedCount)
}

type Article struct {
	Id       int64  `bson:"id,omitempty"`
	Title    string `bson:"title,omitempty"`
	Content  string `bson:"content,omitempty"`
	AuthorId int64  `bson:"author_id,omitempty"`
	Status   uint8  `bson:"status,omitempty"`
	Ctime    int64  `bson:"ctime,omitempty"`
	Utime    int64  `bson:"utime,omitempty"`
}
