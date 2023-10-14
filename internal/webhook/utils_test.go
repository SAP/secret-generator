/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and secret-generator contributors
SPDX-License-Identifier: Apache-2.0
*/

package webhook

import (
	"testing"
)

func TestNormalizeSymbols(t *testing.T) {
	if symbols := normalizeSymbols(Symbols); symbols != Symbols {
		t.Error("normalizeSymbols: got invalid symbols")
	}

	if symbols := normalizeSymbols("_+-_+-"); symbols != "-_+" {
		t.Error("normalizeSymbols: got invalid symbols")
	}
}
