-- Back to one recipe per quotation: the first line's recipe. The other lines are lost (the total,
-- which already includes them and the delivery, stays).
UPDATE quotations q SET recipe_id = l.recipe_id
FROM (SELECT DISTINCT ON (quotation_id) quotation_id, recipe_id FROM quotation_lines ORDER BY quotation_id, position) l
WHERE l.quotation_id = q.id;
DROP TABLE quotation_lines;
ALTER TABLE quotations DROP COLUMN delivery_cost;
ALTER TABLE quotations ALTER COLUMN recipe_id SET NOT NULL;
UPDATE quotations SET margin_pct = COALESCE(margin_pct, 0);
ALTER TABLE quotations ALTER COLUMN margin_pct SET NOT NULL;
