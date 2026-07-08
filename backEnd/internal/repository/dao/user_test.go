package dao

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestGORMUserDAO_Insert(t *testing.T) {
	testCases := []struct {
		//测试用例名称
		name string

		//上下文控制，作为参数传入
		ctx context.Context
		//用户实体，Insert方法需要传入的参数
		u User

		//行为模拟器，隔离真实依赖（MySQL）
		mock func(t *testing.T) *sql.DB

		//预期返回的错误类型
		expectedErr error
	}{
		{
			//测试用例名称
			name: "插入成功～",
			ctx:  context.Background(),
			//构造传入 Insert 方法的用户对象
			u: User{
				Email: sql.NullString{
					String: "123@qq.com",
					Valid:  true,
				},
			},
			mock: func(t *testing.T) *sql.DB {
				//1.创建模拟链接
				//创建一个模拟的 *sql.DB 及其配套的 mock 对象。
				mockDB, mock, err := sqlmock.New()
				//2.设定期望，执行一条INSERT SQL
				//用于匹配将要执行的 SQL 语句（支持正则），并预设返回结果
				//（成功时返回影响行数和自增ID；失败时返回自定义错误）。
				mock.ExpectExec("INSERT INTO `users` .*").
					//3.SQL被执行后，返回一个成功结果，自增ID=3，影响行数=1
					WillReturnResult(sqlmock.NewResult(3, 1))
				//4.确保模拟对象创建成功（一般不会失败）
				require.NoError(t, err)
				//5.返回mock的*sql.DB，供GORM使用。
				return mockDB
			},
			expectedErr: nil,
		},
		{
			name: "邮箱/手机号已存在！",
			ctx:  context.Background(),
			u:    User{},
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnError(&mysql.MySQLError{
						Number: 1062,
					})
				require.NoError(t, err)
				return mockDB
			},
			expectedErr: ErrUserDuplicated,
		},
		{
			name: "数据库错误！",
			ctx:  context.Background(),
			u:    User{},
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnError(errors.New("数据库错误！"))
				require.NoError(t, err)
				return mockDB
			},
			expectedErr: errors.New("数据库错误！"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			db, err := gorm.Open(gormMysql.New(
				gormMysql.Config{
					//直接注入已有的 *sql.DB，绕过 GORM 默认的数据库连接过程。
					Conn: tc.mock(t),
					//避免 GORM 额外查询版本信息，加速测试。
					SkipInitializeWithVersion: true,
				}),
				&gorm.Config{
					DisableAutomaticPing: false,
					//禁用默认事务，让 Create 直接执行，简化模拟。
					SkipDefaultTransaction: true,
				})
			//func NewUserDAO(db *gorm.DB) UserDAO
			d := NewUserDAO(db)
			err = d.Insert(tc.ctx, tc.u)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
