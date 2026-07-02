#Example 1
Phase 3 stock-out doesn't block negative stock — the form accepts qty 999
on a product with 5. Fix it in the service layer and add a test for it.
#Eample 2
Split adjustments into movement_type 'adjustment_in' / 'adjustment_out'
(or add a signed delta column). Update the movement report and CSV so
each row is self-describing. Add a migration; don't rewrite history rows,
backfill direction from quantity_after.