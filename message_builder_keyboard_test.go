package telebot

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/gomodule/redigo/redis"
)

type keyboardProbe struct{}

func (*keyboardProbe) ToTelegramButton(orm.DB, TelegramButtonConversionArgs) InlineButton {
	return InlineButton{Text: "row"}
}

func TestCreateGenericKeyboard_EmptyBodyKeepsMergeButtons(t *testing.T) {
	db := openKeyboardTestDB(t)
	query := db.Model().TableExpr("(SELECT 1 WHERE FALSE) AS empty_body")

	builder := &MessageBuilder{}
	CreateGenericKeyboard[*keyboardProbe](
		builder,
		query,
		&keyboardRedis{},
		db,
		"",
		CreateGenericKeyboardParams{
			ChatID:         42,
			PageUnique:     "last_seen_list",
			ShowNavigation: true,
			MergeButtons: [][]InlineButton{{
				{Text: "Добавить пользователя", Data: "last_seen_add"},
			}},
		},
	)

	message := builder.Build(42)
	rows := message.ReplyMarkup.InlineKeyboard
	if len(rows) != 1 || len(rows[0]) != 1 {
		t.Fatalf("keyboard = %#v, want one merge row", rows)
	}
	if rows[0][0].Text != "Добавить пользователя" || rows[0][0].Data != "last_seen_add" {
		t.Fatalf("button = %#v", rows[0][0])
	}
}

func TestCreateGenericKeyboard_EmptyBodyWithoutMergeButtons(t *testing.T) {
	db := openKeyboardTestDB(t)
	query := db.Model().TableExpr("(SELECT 1 WHERE FALSE) AS empty_body")

	builder := &MessageBuilder{}
	CreateGenericKeyboard[*keyboardProbe](
		builder,
		query,
		&keyboardRedis{},
		db,
		"",
		CreateGenericKeyboardParams{
			ChatID:         42,
			PageUnique:     "last_seen_list",
			ShowNavigation: true,
		},
	)

	message := builder.Build(42)
	if len(message.ReplyMarkup.InlineKeyboard) != 0 {
		t.Fatalf("keyboard = %#v, want empty", message.ReplyMarkup.InlineKeyboard)
	}
}

func openKeyboardTestDB(t *testing.T) *pg.DB {
	t.Helper()
	for _, addr := range []string{"127.0.0.1:43211", "127.0.0.1:5432"} {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err != nil {
			continue
		}
		_ = conn.Close()

		db := pg.Connect(&pg.Options{
			Addr:        addr,
			User:        "chatdetective",
			Password:    "chatdetective",
			Database:    "chatdetective",
			DialTimeout: time.Second,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err = db.ExecContext(ctx, "SELECT 1")
		cancel()
		if err != nil {
			_ = db.Close()
			continue
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	}
	t.Skip("postgres is not available")
	return nil
}

type keyboardRedis struct{}

func (keyboardRedis) Close() error                      { return nil }
func (keyboardRedis) Err() error                        { return nil }
func (keyboardRedis) Send(string, ...interface{}) error { return nil }
func (keyboardRedis) Flush() error                      { return nil }
func (keyboardRedis) Receive() (interface{}, error)     { return nil, nil }

func (keyboardRedis) Do(cmd string, args ...interface{}) (interface{}, error) {
	switch cmd {
	case "HGET":
		return nil, redis.ErrNil
	case "HSET", "EXPIRE":
		return int64(1), nil
	default:
		return nil, redis.ErrNil
	}
}
