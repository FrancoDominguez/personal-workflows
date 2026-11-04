package utils

import (
	"context"
	"errors"
	"log"

	"cloud.google.com/go/firestore"
)

func GetItem[T any](ctx context.Context, client *firestore.Client, collection string, document string, field string) (T, error) {
	var err error
	var result T
	var snapshot *firestore.DocumentSnapshot
	var fieldValue any
	var ErrWrongType = errors.New("found field has wrong type")

	doc := client.Collection(collection).Doc(document)

	snapshot, err = doc.Get(ctx)
	if err != nil {
		log.Printf("fetching cursor document from firestore: %v", err)
		return result, err
	}

	fieldValue, err = snapshot.DataAt(field)
	if err != nil {
		log.Printf("reading document field: %v", err)
		return result, err
	}
	result, ok := fieldValue.(T)
	if !ok {
		log.Printf("%s is not a %T, got %T", field, result, fieldValue)
		return result, ErrWrongType
	}

	return result, nil
}

func SetItem(ctx context.Context, client *firestore.Client, collection string, document string, field string, value any) error {
	doc := client.Collection(collection).Doc(document)

	_, err := doc.Set(ctx, map[string]any{field: value}, firestore.MergeAll)
	if err != nil {
		log.Printf("writing %s.%s field %q: %v", collection, document, field, err)
		return err
	}

	return nil
}
