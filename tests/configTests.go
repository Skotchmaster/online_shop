package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Skotchmaster/online_shop/internal/handlers"
	authhdl "github.com/Skotchmaster/online_shop/internal/handlers/auth"
	"github.com/Skotchmaster/online_shop/internal/handlers/cart"
	"github.com/Skotchmaster/online_shop/internal/hash"
	"github.com/Skotchmaster/online_shop/internal/models"
	"github.com/Skotchmaster/online_shop/internal/mykafka"
	"github.com/labstack/echo/v4"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testEnv struct {
	T                        *testing.T
	E                        *echo.Echo
	A                        *authhdl.AuthHandler
	C                        *cart.CartHandler
	P                        *handlers.ProductHandler
	DB                       *gorm.DB
	JWTSecret, RefreshSecret []byte
}

func InitTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"postgres", "root", "dbtest", "5432", "test_db",
	)
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	return db
}

func (env *testEnv) ClearDB() {

	tables := []string{
		"order_items",
		"orders",
		"cart_items",
		"refresh_tokens",
		"users",
		"products",
	}

	query := fmt.Sprintf(
		"TRUNCATE TABLE %s RESTART IDENTITY CASCADE",
		strings.Join(tables, ", "),
	)
	env.DB.Exec(query)
}

func LoadConfig(t *testing.T) (*gorm.DB, []byte, []byte) {
	db := InitTestDB(t)

	jwt_secret := []byte(os.Getenv("JWT_SECRET"))
	refresh := []byte(os.Getenv("REFRESH_SECRET"))

	return db, jwt_secret, refresh
}

func WaitForKafkaTopic(t *testing.T, topic string) {
	conn, err := kafka.Dial("tcp", "kafka:9092")
	if err != nil {
		t.Fatalf("Failed to connect to Kafka: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 10; i++ {
		partitions, err := conn.ReadPartitions(topic)
		if err == nil && len(partitions) > 0 {
			return
		}
		t.Logf("Topic %s not found (attempt %d)", topic, i+1)
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("Topic %s not created after 20 seconds", topic)
}

func newTestEnv(t *testing.T) *testEnv {
	db, jwt, refresh := LoadConfig(t)

	ensureTopics(t, "kafka:9092", "user_events", "cart_events", "product_events")

	WaitForKafkaTopic(t, "user_events")
	WaitForKafkaTopic(t, "cart_events")
	WaitForKafkaTopic(t, "product_events")
	prod, err := mykafka.NewProducer(
		[]string{"kafka:9092"},
		[]string{"user_events", "cart_events", "product_events"},
	)
	if err != nil {
		t.Fatalf("Failed to create Kafka producer: %v", err)
	}

	env := &testEnv{
		T:             t,
		E:             echo.New(),
		DB:            db,
		JWTSecret:     jwt,
		RefreshSecret: refresh,
	}

	env.ClearDB()
	env.A = &authhdl.AuthHandler{
		DB:            db,
		JWTSecret:     jwt,
		RefreshSecret: refresh,
		Producer:      prod,
	}

	env.C = &cart.CartHandler{
		DB:        db,
		JWTSecret: jwt,
		Producer:  prod,
	}

	env.P = &handlers.ProductHandler{
		DB:        db,
		JWTSecret: jwt,
		Producer:  prod,
	}

	t.Cleanup(func() {
		env.ClearDB()
		prod.Close()
	})

	return env
}

func consumeNextEvent(t *testing.T, topic string, produce func()) map[string]interface{} {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	conn, err := kafka.DialLeader(ctx, "tcp", "kafka:9092", topic, 0)
	require.NoError(t, err)
	end, err := conn.ReadLastOffset()
	require.NoError(t, err)
	_ = conn.Close()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{"kafka:9092"},
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
		MaxWait:   time.Second,
	})
	defer r.Close()
	require.NoError(t, r.SetOffset(end))

	produce()

	m, err := r.ReadMessage(ctx)
	require.NoError(t, err)

	var event map[string]interface{}
	require.NoError(t, json.Unmarshal(m.Value, &event))
	return event
}

func ensureTopics(t *testing.T, broker string, topics ...string) {
	t.Helper()

	conn, err := kafka.Dial("tcp", broker)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	admin, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer admin.Close()

	var cfgs []kafka.TopicConfig
	for _, tp := range topics {
		cfgs = append(cfgs, kafka.TopicConfig{
			Topic:             tp,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	err = admin.CreateTopics(cfgs...)
	if err != nil && !strings.Contains(err.Error(), "Topic with this name already exists") {
		require.NoError(t, err)
	}
}

func (env *testEnv) doJSONRequest(method, path string, body interface{}, cookies ...*http.Cookie) (*httptest.ResponseRecorder, []byte, echo.Context) {
	var buf bytes.Buffer
	if body != nil {
		require.NoError(env.T, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	c := env.E.NewContext(req, rec)
	return rec, rec.Body.Bytes(), c
}

func login(t *testing.T, env *testEnv) (string, string) {
	t.Helper()

	creds := map[string]string{
		"username": "admin",
		"password": "admin",
	}

	rec, _, c := env.doJSONRequest(http.MethodPost, "/login", creds)
	require.NoError(t, env.A.Login(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var accessToken, refreshToken string
	for _, ck := range rec.Result().Cookies() {
		switch ck.Name {
		case "accessToken":
			accessToken = ck.Value
		case "refreshToken":
			refreshToken = ck.Value
		}
	}
	require.NotEmpty(t, accessToken, "accessToken cookie must be set")
	require.NotEmpty(t, refreshToken, "refreshToken cookie must be set")

	return accessToken, refreshToken
}

func login_admin(t *testing.T, env *testEnv) (string, string) {
	t.Helper()

	pwdHash, err := hash.HashPassword("test_password")
	require.NoError(t, err)

	u := models.User{
		Username:     "test_user",
		PasswordHash: pwdHash,
		Role:         "admin",
	}

	var existing models.User
	if err := env.DB.Where("username = ?", u.Username).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			require.NoError(t, env.DB.Create(&u).Error)
		} else {
			require.NoError(t, err)
		}
	} else {
		existing.PasswordHash = u.PasswordHash
		existing.Role = u.Role
		require.NoError(t, env.DB.Save(&existing).Error)
	}

	creds := map[string]string{
		"username": u.Username,
		"password": "test_password",
	}

	rec, _, c := env.doJSONRequest(http.MethodPost, "/login", creds)
	require.NoError(t, env.A.Login(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var accessToken, refreshToken string
	for _, ck := range rec.Result().Cookies() {
		switch ck.Name {
		case "accessToken":
			accessToken = ck.Value
		case "refreshToken":
			refreshToken = ck.Value
		}
	}
	require.NotEmpty(t, accessToken, "accessToken cookie must be set")
	require.NotEmpty(t, refreshToken, "refreshToken cookie must be set")

	return accessToken, refreshToken
}

