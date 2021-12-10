/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package table

import "fmt"

type Header struct {
	Name      string
	MaxLength int
}

type Printer struct {
	Headers          []Header
	RowConverterFunc func(obj interface{}) []string
}

func (p *Printer) Print(objects []interface{}) {
	rowList := p.generateRowList(objects)

	columnLengths := p.findColumnLengths(rowList)
	template := templateFromLengths(columnLengths)

	for _, row := range rowList {
		rowInterfaces := make([]interface{}, len(row))
		for i := range row {
			rowInterfaces[i] = row[i]
		}

		fmt.Printf(template, rowInterfaces...)
	}
}

func (p *Printer) generateRowList(objects []interface{}) [][]string {
	headerRow := []string{}
	for _, header := range p.Headers {
		headerRow = append(headerRow, header.Name)
	}

	rowList := [][]string{headerRow}

	for _, row := range objects {
		rowList = append(rowList, p.RowConverterFunc(row))
	}

	return rowList
}

func templateFromLengths(columnLengths []int) string {
	sprintfTemplate := ""
	for _, length := range columnLengths {
		sprintfTemplate += fmt.Sprintf("%%-%d.%ds", length+2, length)
	}

	return sprintfTemplate + "\n"
}

func (p *Printer) findColumnLengths(rowList [][]string) []int {
	columns := len(rowList[0])
	columnLengths := make([]int, columns)

	for _, row := range rowList {
		for index, column := range row {
			colLength := len(column)

			// trim the column length if it's going over our maximum
			if colLength > p.Headers[index].MaxLength {
				colLength = p.Headers[index].MaxLength
			}

			if colLength > columnLengths[index] {
				columnLengths[index] = colLength
			}
		}
	}

	return columnLengths
}
