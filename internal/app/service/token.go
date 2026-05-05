package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xvq/go-template/internal/core"
)

const (
	redisKeyName    = "token:data"
	redisMapKeyName = "token:maps"
	defaultExpires  = 15 * 86400
)

type TokenData struct {
	UID     uint   `json:"uid"`
	Expires int64  `json:"expires"`
	Model   string `json:"model"`
}

func Generate(uid uint, model string, singleDevice bool) string {
	if core.Redis == nil {
		return ""
	}

	ctx := context.Background()
	token := generateToken()
	mapKey := model + "-" + itoa(uid)

	mapData := getTokenMaps(ctx, mapKey)

	if singleDevice {
		for _, t := range mapData {
			core.Redis.HDel(ctx, redisKeyName, t)
		}
		mapData = []string{token}
	} else {
		mapData = append(mapData, token)
	}

	saveMapData, _ := json.Marshal(mapData)
	core.Redis.HSet(ctx, redisMapKeyName, mapKey, string(saveMapData))

	tokenData := TokenData{
		UID:     uid,
		Expires: time.Now().Unix() + defaultExpires,
		Model:   model,
	}
	tdBytes, _ := json.Marshal(tokenData)
	core.Redis.HSet(ctx, redisKeyName, token, string(tdBytes))

	return token
}

func Validate(token string) *TokenData {
	if core.Redis == nil || token == "" {
		return nil
	}

	ctx := context.Background()
	data, err := core.Redis.HGet(ctx, redisKeyName, token).Result()
	if err != nil {
		return nil
	}

	var td TokenData
	if err := json.Unmarshal([]byte(data), &td); err != nil {
		return nil
	}

	if time.Now().Unix() > td.Expires {
		core.Redis.HDel(ctx, redisKeyName, token)
		removeTokenMap(ctx, td.Model, td.UID, token)
		return nil
	}

	return &td
}

func RevokeByToken(token string) bool {
	if core.Redis == nil || token == "" {
		return false
	}

	ctx := context.Background()
	data, err := core.Redis.HGet(ctx, redisKeyName, token).Result()
	if err != nil {
		return false
	}

	core.Redis.HDel(ctx, redisKeyName, token)

	var td TokenData
	json.Unmarshal([]byte(data), &td)
	removeTokenMap(ctx, td.Model, td.UID, token)

	return true
}

func RevokeByUid(model string, uid uint) bool {
	if core.Redis == nil {
		return false
	}

	ctx := context.Background()
	mapKey := model + "-" + itoa(uid)

	tokenMaps, err := core.Redis.HGet(ctx, redisMapKeyName, mapKey).Result()
	if err != nil {
		return false
	}

	var tokens []string
	json.Unmarshal([]byte(tokenMaps), &tokens)

	for _, t := range tokens {
		core.Redis.HDel(ctx, redisKeyName, t)
	}
	core.Redis.HDel(ctx, redisMapKeyName, mapKey)

	return true
}

func CleanExpired() int {
	if core.Redis == nil {
		return 0
	}

	ctx := context.Background()
	allTokens, err := core.Redis.HGetAll(ctx, redisKeyName).Result()
	if err != nil {
		return 0
	}

	count := 0
	now := time.Now().Unix()

	for token, data := range allTokens {
		var td TokenData
		if err := json.Unmarshal([]byte(data), &td); err != nil {
			continue
		}
		if now > td.Expires {
			core.Redis.HDel(ctx, redisKeyName, token)
			removeTokenMap(ctx, td.Model, td.UID, token)
			count++
		}
	}

	return count
}

func generateToken() string {
	ctx := context.Background()
	for {
		b := make([]byte, 32)
		rand.Read(b)
		token := hex.EncodeToString(b)
		exists, _ := core.Redis.HExists(ctx, redisKeyName, token).Result()
		if !exists {
			return token
		}
	}
}

func getTokenMaps(ctx context.Context, mapKey string) []string {
	data, err := core.Redis.HGet(ctx, redisMapKeyName, mapKey).Result()
	if err != nil {
		return []string{}
	}
	var result []string
	json.Unmarshal([]byte(data), &result)
	return result
}

func removeTokenMap(ctx context.Context, model string, uid uint, token string) {
	mapKey := model + "-" + itoa(uid)

	tokenMaps, err := core.Redis.HGet(ctx, redisMapKeyName, mapKey).Result()
	if err != nil {
		return
	}

	var tokens []string
	json.Unmarshal([]byte(tokenMaps), &tokens)

	newTokens := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t != token {
			newTokens = append(newTokens, t)
		}
	}

	if len(newTokens) > 0 {
		saveData, _ := json.Marshal(newTokens)
		core.Redis.HSet(ctx, redisMapKeyName, mapKey, string(saveData))
	} else {
		core.Redis.HDel(ctx, redisMapKeyName, mapKey)
	}
}

func itoa(n uint) string {
	return fmt.Sprint(n)
}
