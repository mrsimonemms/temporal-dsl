/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package utils

import (
	"os"
	"strings"
)

// LoadEnvvars load all environment variables that start with the prefix. Removes prefix for storage
func LoadEnvvars(prefix string) map[string]any {
	vars := map[string]any{}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		key := pair[0]
		value := pair[1]

		if strings.HasPrefix(key, prefix) {
			// Remove the prefix from the key
			vars[strings.TrimLeft(key, prefix)] = value
		}
	}

	return vars
}
