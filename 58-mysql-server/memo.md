## プロダクトのリスト

```sql
SELECT DISTINCT
  p.part_no,
  p.part_name
FROM t_bom b
JOIN m_part p
  ON p.part_id = b.parent_part_id
WHERE p.part_name LIKE 'Product-%'
ORDER BY p.part_name;
```

## プロダクトを指定して、下位階層まで全部展開したい。

```sql
WITH RECURSIVE bom_tree AS (
  SELECT
    1 AS lvl,
    parent.part_id AS root_part_id,
    parent.part_name AS root_part_name,
    child.part_id AS child_part_id,
    child.part_no AS child_part_no,
    child.part_name AS child_part_name,
    bi.qty_per_parent,
    bi.qty_per_parent AS total_qty
  FROM m_part parent
  JOIN t_bom b ON b.parent_part_id = parent.part_id
  JOIN t_bom_item bi ON bi.bom_id = b.bom_id
  JOIN m_part child ON child.part_id = bi.child_part_id
  WHERE parent.part_name = 'Product-39999'

  UNION ALL

  SELECT
    bt.lvl + 1,
    bt.root_part_id,
    bt.root_part_name,
    c2.part_id,
    c2.part_no,
    c2.part_name,
    bi2.qty_per_parent,
    bt.total_qty * bi2.qty_per_parent
  FROM bom_tree bt
  JOIN t_bom b2 ON b2.parent_part_id = bt.child_part_id
  JOIN t_bom_item bi2 ON bi2.bom_id = b2.bom_id
  JOIN m_part c2 ON c2.part_id = bi2.child_part_id
)
SELECT
  lvl,
  child_part_no,
  child_part_name,
  qty_per_parent,
  total_qty
FROM bom_tree
ORDER BY lvl, child_part_no;
```




cd /home/ubuntu/marmot-manifests/58-mysql-server
. .venv/bin/activate
python stress_bomdb_transactions.py \
  --host 10.1.1.10 \
  --user mysqladmin \
  --password komekomekome \
  --database bomdb \
  --workers 24 \
  --duration 300 \
  --rollback-ratio 0.3 \
  --create-table