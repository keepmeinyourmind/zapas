// Package profile — что система знает о доходе клиента: обычный доход, уровень доверия,
// регулярные платежи. Чистые функции: без БД и без time.Now().
package profile

import "github.com/keepmeinyourmind/zapas/src/internal/domain"

// Trust — уровень доверия по числу месяцев истории дохода:
// меньше 3 — Low (всегда спрашиваем), 3–5 — Mid, 6 и больше — High.
func Trust(monthsOfData int) domain.TrustLevel {
	switch {
	case monthsOfData < 3:
		return domain.TrustLevelLow
	case monthsOfData < 6:
		return domain.TrustLevelMid
	default:
		return domain.TrustLevelHigh
	}
}
