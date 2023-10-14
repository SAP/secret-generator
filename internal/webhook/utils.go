/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook

import "strings"

func normalizeSymbols(symbols string) string {
	var l []rune
	var i = 0
	for _, r := range Symbols {
		if strings.ContainsRune(symbols, r) {
			l = append(l, r)
			i++
		}
	}
	return string(l)
}
