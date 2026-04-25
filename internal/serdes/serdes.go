// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serdes

import "reflect"

func Serialize(data any) ([]byte, error) {
	return resolveCodecCaching(reflect.TypeOf(data)).encode(encodeCtx{}, []byte{}, reflect.ValueOf(data))
}

func Deserialize(data []byte, res any) error {
	_, err := resolveCodecCaching(reflect.TypeOf(res)).decode(decodeCtx{}, data, reflect.ValueOf(res))
	return err
}
