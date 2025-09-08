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

package dsl

import "time"

type (
	ErrType    string
	ResultType string
)

const (
	CallHTTPErr    ErrType = "CallHTTP error"
	IfStatementErr ErrType = "IfStatement error"
)

const (
	CallHTTPResultType             ResultType = "CallHTTP"
	ForkResultType                 ResultType = "Fork"
	ListenResultType               ResultType = "Listen"
	CallRunChildWorkflowResultType ResultType = "RunChildWorkflow"
	SetResultType                  ResultType = "Set"
)

const defaultWorkflowTimeout = time.Minute * 5

const MetadataSearchAttribute = "searchAttributes"

const (
	SearchAttributeDateTimeType    string = "datetime"
	SearchAttributeKeywordListType string = "keywordlist"
	SearchAttributeKeywordType     string = "keyword"
	SearchAttributeTextType        string = "text"
	SearchAttributeIntType         string = "int"
	SearchAttributeDoubleType      string = "double"
	SearchAttributeBooleanType     string = "bool"
)
