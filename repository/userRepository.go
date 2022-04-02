package repository

import (
	"context"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"messageService/model"
	"strconv"
)

type UserRepository struct {
	DB *pgx.Conn
}

func (r *UserRepository) FindByIDs(uIds []string) ([]*model.UserModel, error) {
	var users []*model.UserModel
	ids := make([]interface{}, len(uIds))
	ids[0] = uIds[0]
	inStmt := "$1"
	for i := 1; i < len(uIds); i++ {
		inStmt = inStmt + ",$" + strconv.Itoa(i+1)
		ids[i] = uIds[i]
	}
	if err := pgxscan.Select(context.Background(), r.DB, &users, "SELECT * FROM user_account WHERE uid IN ("+inStmt+")", ids...); err != nil {
		return nil, err
	}
	return users, nil
}
