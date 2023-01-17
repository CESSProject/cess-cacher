package service

import (
	"cess-cacher/base/cache"
	resp "cess-cacher/server/response"
	"cess-cacher/utils"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

var key string
var ValidDuration = time.Minute * 30

type Claims struct {
	Ticket Ticket `json:"ticket,omitempty"`
	jwt.RegisteredClaims
}

type AuthReq struct {
	BID  string `json:"bid"`
	Sign []byte `json:"sign"`
}

func GenerateToken(bid string, sign []byte) (string, resp.Error) {
	var stoken string
	//check order
	t, err := PraseTicketByBID(bid)
	if err != nil {
		return stoken, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	if !utils.VerifySign(t.Account, []byte(bid), sign) {
		return stoken, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	//data preheating: prepare the files not downloaded
	cache.GetCacheHandle().HitOrLoad(t.FileHash)
	if key == "" {
		key = utils.GetRandomcode(32)
	}
	now := time.Now()
	claims := Claims{
		Ticket: t,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now.Add(-30)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ValidDuration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	stoken, err = token.SignedString(key)
	if err != nil {
		return stoken, resp.NewError(500, errors.Wrap(err, "generate token error"))
	}
	return stoken, nil
}

func PraseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("token invalid")
}

func ReFreshToken(tokenStr string) (string, error) {
	jwt.TimeFunc = func() time.Time {
		return time.Unix(0, 0)
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		jwt.TimeFunc = time.Now
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(ValidDuration))
		return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
	}
	return "", nil
}
