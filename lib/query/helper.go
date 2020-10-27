/*
 * Copyright 2018 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package query

import (
	"errors"
	"log"
	"reflect"
)

func InterfaceSlice(slice interface{}) (result []interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: Recovered in InterfaceSlice", r)
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		err = errors.New("unable to interpret value as slice in InterfaceSlice")
		return
	}

	result = make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		result[i] = s.Index(i).Interface()
	}
	return
}

func InterfaceSliceRemove(slice interface{}, value interface{}) (result []interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: Recovered in InterfaceSliceRemove", r)
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		err = errors.New("unable to interpret value as slice in InterfaceSliceRemove")
		return
	}

	result = []interface{}{}

	for i := 0; i < s.Len(); i++ {
		element := s.Index(i).Interface()
		if !reflect.DeepEqual(element, value) {
			result = append(result, element)
		}
	}
	return
}

func InterfaceSliceAppend(slice interface{}, value interface{}) (result []interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("ERROR: Recovered in InterfaceSliceAppend", r)
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		err = errors.New("unable to interpret value as slice in InterfaceSliceAppend")
		return
	}

	for i := 0; i < s.Len(); i++ {
		result = append(result, s.Index(i).Interface())
	}

	result = append(result, value)

	return
}
