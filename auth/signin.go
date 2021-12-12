package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/containerish/OpenRegistry/types"
	"github.com/labstack/echo/v4"
)

func (a *auth) SignIn(ctx echo.Context) error {
	ctx.Set(types.HandlerStartTime, time.Now())
	defer func() {
		a.logger.Log(ctx).Send()
	}()
	var user User

	if err := json.NewDecoder(ctx.Request().Body).Decode(&user); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": err.Error(),
		})
	}
	if user.Email == "" && user.Username == "" {
		errMsg := echo.Map{
			"error": "email and username cannot be empty, please provide at least one of them",
		}
		ctx.Set(types.HttpEndpointErrorKey, errMsg)
		return ctx.JSON(http.StatusBadRequest, errMsg)
	}

	if user.Password == "" {
		errMsg := echo.Map{
			"error": "password cannot be empty",
		}
		ctx.Set(types.HttpEndpointErrorKey, errMsg)
		return ctx.JSON(http.StatusBadRequest, errMsg)
	}

	var key string

	if user.Email != "" {
		if err := verifyEmail(user.Email); err != nil {
			ctx.Set(types.HttpEndpointErrorKey, err.Error())
			return ctx.JSON(http.StatusBadRequest, echo.Map{
				"error": err.Error(),
			})
		}
		key = fmt.Sprintf("%s/%s", UserNameSpace, user.Email)
	} else {
		key = fmt.Sprintf("%s/%s", UserNameSpace, user.Username)
	}

	bz, err := a.store.Get([]byte(key))
	if err != nil {
		ctx.Set(types.HttpEndpointErrorKey, err.Error())
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": err.Error(),
		})
	}

	var userFromDb User
	if err := json.Unmarshal(bz, &userFromDb); err != nil {
		ctx.Set(types.HttpEndpointErrorKey, err.Error())
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	if !a.verifyPassword(userFromDb.Password, user.Password) {
		errMsg := "invalid password"
		ctx.Set(types.HttpEndpointErrorKey, errMsg)
		return ctx.JSON(http.StatusUnauthorized, errMsg)
	}

	tokenLife := time.Now().Add(time.Hour * 24 * 14).Unix()
	token, err := a.newToken(userFromDb, tokenLife)
	if err != nil {
		ctx.Set(types.HttpEndpointErrorKey, err.Error())
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(http.StatusOK, echo.Map{
		"token":      token,
		"expires_in": tokenLife,
		"issued_at":  time.Now().Unix(),
	})

}
