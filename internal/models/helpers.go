package models

import (
	"reflect"

	"github.com/jinzhu/gorm"
)

// runValidationFunctions is a type-agnostic implementation of the validation function
// runner. It is to be reused by the services.
//
// The value parameter is a pointer to the value being validated and it is passed to each validation
// function. The fns parameter must be a list of validation function wrappers that return their field name
// and the actual validation function. The validation function returned must take value as an input and
// return an error. For example:
//
//		type User struct {
//			...
//		}
//
//		type valFunc func(*User) error
//
//		func validatorWrapper() (string, valFunc) {
//			return "fieldName", func (u *User) error {
//				...
//				return FieldErrorValue
//			}
//		}
//
//		func runValidation(u *User, fns ...func() (string, valFunc)) error {
//			return runValidationFunctions(u, fns)
//		}
//
// This pattern must be followed closely, or this function will panic with obscure errors.
//
// If a validator is not related to a specific field, its wrapper must return an empty string as its field name.
//
// Whan a field validator returns an error, that error is stored in a ValidationError value and no other validators
// for the same field are called.
//
// If a field's validator returns another ValidationError, these will be merged with the resulting field errors
// where the field name will be <validator_field>.<returned_validation_error_field>. If a field's validator returns a
// PublicError implementer, it will simply be included in the resulting ValidationError. Otherwise, the function
// returns immediately with the error returned by the validator.
//
// A non-field validator is only executed if no field errors have been returned by previous validators.
func runValidationFunctions(value interface{}, fns interface{}) error {
	var ve = ValidationError{}

	rfns := reflect.ValueOf(fns)
	for i := 0; i < rfns.Len(); i++ {

		rrets := rfns.Index(i).Call(nil)
		field, rfn := rrets[0].Interface().(string), rrets[1]

		// if it's not a field error
		if field == "" {
			// and no other field errors have been registered yet
			if len(ve) == 0 {
				// then run the non-field validator and return straight away in case of error
				rerr := rfn.Call([]reflect.Value{reflect.ValueOf(value)})
				if err := rerr[0].Interface(); err != nil {
					return err.(error)
				}
			}

			continue
		}

		// else if it is a field validator and no errors yet on this field
		if ve[field] == nil {
			rerr := rfn.Call([]reflect.Value{reflect.ValueOf(value)})

			// run the validation function, if it errors...
			if err := rerr[0].Interface(); err != nil {
				switch terr := err.(type) {
				case ValidationError: // and the error is a ValidationError map, merge the maps
					for k, v := range terr {
						ve[field+"."+k] = v
					}

				case PublicError: // and the error is a PublicError, put it in the ValidationError map
					ve[field] = terr

				default: // otherwise, it's a private error so we should exit
					return terr.(error)
				}
			}
		}
	}

	if len(ve) > 0 {
		return ve
	}

	return nil
}

// gormToMap helps us get around the issue where stupid GORM is unable to simply and only run
// an UPDATE for all fields of an object regardless of their value. We instead need to pass a
// map to db.Updates(...) to have all fields updated.
func gormToMap(db *gorm.DB, obj interface{}) map[string]interface{} {
	fields := db.NewScope(obj).Fields()
	ret := make(map[string]interface{}, len(fields))

	for _, f := range fields {
		ret[f.DBName] = f.Field.Interface()
	}

	return ret
}

// gormTransaction wraps the given function in a transaction. In case the given
// functions returns an error, the transaction will be rolled back.
func gormTransaction(db *gorm.DB, f func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	err := f(tx)
	if err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			return wrapi("gorm transaction rollback error", rbErr)
		}
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return wrapi("gorm transaction commit error", err)
	}

	return nil
}
