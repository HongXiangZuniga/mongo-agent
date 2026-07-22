// Package authmongo es el adapter de salida hacia la base de usuarios del login.
//
// Está SEPARADO de pkg/persistence/mongodb a propósito: ese paquete tiene el
// invariante de solo lectura del agente, verificado con
// mongodb.VerifyReadOnlyGuarantee sobre el cluster Atlas. Esta base es una
// instancia MongoDB local dedicada exclusivamente a usuarios del login. Este
// repositorio solo LEE usuarios; nunca escribe, actualiza ni borra.
package authmongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/HongXiangZuniga/mongo-agent/pkg/auth"
)

type userDoc struct {
	Username     string `bson:"username"`
	PasswordHash string `bson:"password_hash"`
}

type repository struct {
	coll         *mongo.Collection
	queryTimeout time.Duration
}

// NewUserRepository construye el adapter auth.UserRepository sobre la colección
// de usuarios de la base de login.
func NewUserRepository(db *mongo.Database, collectionName string, queryTimeout time.Duration) auth.UserRepository {
	return &repository{coll: db.Collection(collectionName), queryTimeout: queryTimeout}
}

func (r *repository) FindByUsername(ctx context.Context, username string) (auth.User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var doc userDoc
	err := r.coll.FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, err
	}
	return auth.User{Username: doc.Username, PasswordHash: doc.PasswordHash}, nil
}
