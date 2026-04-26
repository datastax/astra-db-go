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
	v := reflect.ValueOf(data)
	t := v.Type()

	c, err := resolveCodecCaching(t, seenStructs{}, v.CanAddr())
	if err != nil {
		return nil, err
	}

	return c.encode(encodeCtx{}, []byte{}, v)
}

func Deserialize(data []byte, res any) error {
	v := reflect.ValueOf(res)
	t := v.Type()

	c, err := resolveCodecCaching(t, seenStructs{}, v.CanAddr())
	if err != nil {
		return err
	}

	_, err = c.decode(decodeCtx{}, data, v)
	return err
}
