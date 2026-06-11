package fixture

import (
	"context"

	chaos "github.com/ag4r/chaotic/chaos"
)

const namedConst = "order.beforeShip"

func Run(ctx context.Context, dyn string) {
	_ = chaos.Point(ctx, "checkout.afterCommit")   // literal
	_ = chaos.Point(ctx, namedConst)               // const-folded literal
	_ = chaos.PointWith(ctx, "payment.retry", nil) // literal via PointWith
	_ = chaos.Point(ctx, dyn)                      // dynamic (not gatable)
}
