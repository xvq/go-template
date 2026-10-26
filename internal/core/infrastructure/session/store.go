package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalid = errors.New("invalid session")
var realmPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type Data struct {
	Realm     string `json:"realm"`
	SubjectID uint64 `json:"subject_id"`
	CreatedAt int64  `json:"created_at"` // Unix seconds (UTC).
	ExpiresAt int64  `json:"expires_at"` // Unix seconds (UTC).
}
type Store struct {
	r redis.UniversalClient
}

func New(r redis.UniversalClient) *Store {
	return &Store{r: r}
}
func Hash(token string) string                 { v := sha256.Sum256([]byte(token)); return hex.EncodeToString(v[:]) }
func (s *Store) sessionKey(hash string) string { return "auth:session:" + hash }
func (s *Store) indexKey(realm string, id uint64) string {
	return fmt.Sprintf("auth:index:%s:%d", realm, id)
}

// Standalone Redis / Sentinel only. All related mutations run atomically.
var issueScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 1 then return 0 end
local now = tonumber(ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', now)
if ARGV[4] == '0' then
 local old = redis.call('ZRANGE', KEYS[2], 0, -1)
 for _, hash in ipairs(old) do redis.call('DEL', ARGV[5] .. hash) end
 redis.call('DEL', KEYS[2])
end
redis.call('SET', KEYS[1], ARGV[3], 'EXAT', ARGV[2])
redis.call('ZADD', KEYS[2], ARGV[2], ARGV[6])
local last = redis.call('ZRANGE', KEYS[2], -1, -1, 'WITHSCORES')
redis.call('EXPIREAT', KEYS[2], last[2])
return 1
`)

func (s *Store) Issue(ctx context.Context, realm string, id uint64, ttl time.Duration, multiple bool) (string, *Data, error) {
	if !realmPattern.MatchString(realm) || id == 0 || ttl < time.Second {
		return "", nil, fmt.Errorf("invalid session arguments")
	}
	now := time.Now().Unix()
	d := &Data{Realm: realm, SubjectID: id, CreatedAt: now, ExpiresAt: now + int64(ttl/time.Second)}
	raw, err := json.Marshal(d)
	if err != nil {
		return "", nil, err
	}
	allow := 0
	if multiple {
		allow = 1
	}
	for attempts := 0; attempts < 3; attempts++ {
		b := make([]byte, 32)
		if _, err = rand.Read(b); err != nil {
			return "", nil, err
		}
		token := hex.EncodeToString(b)
		hash := Hash(token)
		n, e := issueScript.Run(ctx, s.r, []string{s.sessionKey(hash), s.indexKey(realm, id)}, now, d.ExpiresAt, string(raw), allow, "auth:session:", hash).Int()
		if e != nil {
			return "", nil, fmt.Errorf("issue session: %w", e)
		}
		if n == 1 {
			return token, d, nil
		}
	}
	return "", nil, fmt.Errorf("session collision")
}
func (s *Store) Validate(ctx context.Context, token string) (*Data, error) {
	if len(token) != 64 {
		return nil, ErrInvalid
	}
	if _, err := hex.DecodeString(token); err != nil {
		return nil, ErrInvalid
	}
	raw, err := s.r.Get(ctx, s.sessionKey(Hash(token))).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	var d Data
	if err = json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("corrupt session data: %w", err)
	}
	if d.SubjectID == 0 || !realmPattern.MatchString(d.Realm) || time.Now().Unix() >= d.ExpiresAt {
		return nil, ErrInvalid
	}
	return &d, nil
}

var revokeScript = redis.NewScript(`
redis.call('DEL', KEYS[1])
redis.call('ZREM', KEYS[2], ARGV[1])
if redis.call('ZCARD', KEYS[2]) == 0 then redis.call('DEL', KEYS[2]) end
return 1
`)

func (s *Store) Revoke(ctx context.Context, token string) error {
	d, err := s.Validate(ctx, token)
	if errors.Is(err, ErrInvalid) {
		return nil
	}
	if err != nil {
		return err
	}
	hash := Hash(token)
	return revokeScript.Run(ctx, s.r, []string{s.sessionKey(hash), s.indexKey(d.Realm, d.SubjectID)}, hash).Err()
}

var revokeAllScript = redis.NewScript(`
local hashes = redis.call('ZRANGE', KEYS[1], 0, -1)
for _, hash in ipairs(hashes) do redis.call('DEL', ARGV[1] .. hash) end
redis.call('DEL', KEYS[1])
return #hashes
`)

func (s *Store) RevokeAll(ctx context.Context, realm string, id uint64) error {
	if !realmPattern.MatchString(realm) || id == 0 {
		return fmt.Errorf("invalid subject")
	}
	return revokeAllScript.Run(ctx, s.r, []string{s.indexKey(realm, id)}, "auth:session:").Err()
}
