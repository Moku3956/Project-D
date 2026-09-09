# B+Tree Specification

## Overview

A B+Tree keyed on the PRIMARY KEY. All data is stored in leaf nodes; internal nodes hold only keys and child pointers.

---

## Node structure

### Internal nodes

Holds keys and pointers to child page IDs. For N keys there are N+1 child pointers (the left-child convention).

```
[Child0]  Key1  [Child1]  Key2  [Child2]
```

- `Child_(i-1)` is responsible for values **less than** `Key_i` (the child to the left of Key_i)
- Values greater than or equal to the largest key are handled by `Child_N`. Since this child isn't
  paired with any key, it's stored separately in the page header's `RightmostChild` (see storage/page spec)

Cells are stored as the pair `(Key_i, Child_(i-1))`, i.e. `[composite key][child page ID, 4 bytes]` (see `encodeInternalCell` in cell.go).
Lookup follows this rule (`findChildPageID`):

```
key < Key1        → Child0 (cell0's child)
Key1 <= key < Key2 → Child1 (cell1's child)
...
KeyN <= key        → RightmostChild
```

### Leaf nodes

Holds keys and values (records) inline. Leaf nodes are connected to each other via a linked list (for range scans).

```
[Key0: Record0][Key1: Record1]...
```

The next leaf page ID reuses the same header field that internal nodes use as a child pointer,
`RightmostChild` (page header bytes 20-24, see storage/page spec); on leaf pages this field is
repurposed to store it (`nextLeafID` / `setNextLeafID`).

---

## Operations

### Search

The root page ID is obtained from the file header. Starting from the root, internal nodes are traversed until reaching the target leaf node, which returns the record.

### Insert

If the leaf node has room, the record is inserted directly. If the leaf node is full, the page is split and the parent internal node is updated. If the root itself splits, a new root is created and the root page ID in the file header is updated.

### Delete

Removes the target record from the leaf node. Page merging is omitted from the initial implementation for simplicity.

### Range scan

Efficient range scans are achieved by walking the leaf nodes' linked list.

---

## PRIMARY KEY

The B+Tree's key is the value of the PRIMARY KEY column. Key comparison is performed according to the column's type.

---

## Key format (composite key)

To store multiple tables' data in a single B+Tree file, keys use the following composite format.

```
[tableID: 4bytes BE][type_tag: 1byte][pk_bytes]
```

- `tableID`: an ID identifying the table (auto-assigned by the catalog)
- `type_tag`: a tag indicating the PK's type (`0x01`=INT, `0x02`=VARCHAR)
- `pk_bytes`: the PK's value bytes (8 bytes BE for INT; a 2-byte length + UTF-8 bytes for VARCHAR)

Because the type tag is embedded, the format is self-describing, so no external type information is needed when decoding.

Sort order compares tableID first, then the PK value within the same tableID. This lets multiple tables' records coexist in the same key space, and Scan can be implemented as a prefix range scan on tableID.

---

## Nested cell structure (a concrete example)

When inserting `(id=56, name='Bob')` into a table `(id INT PRIMARY KEY, name VARCHAR(50))`,
the page, slot, cell, and composite key nest as follows.

![alt text](image.png)

```
Page (4096 bytes)
├─ Header (24 bytes)
├─ Slot array
│   └─ 1 slot (4 bytes) = [start offset 2 bytes][length 2 bytes]
│        └─ These two values point to one cell within the cell data (independent of adjacent slots' positions)
└─ Cell data
    └─ 1 cell (leaf node, 31 bytes)
        ├─ Composite key (13 bytes)  ← tableID(4) + type_tag(1) + id value(8)
        ├─ NULL bitmap (1 byte)
        ├─ Offset array (4 bytes)  ← number of columns(2) × 2 bytes
        └─ Column data (13 bytes)  ← id(8) + name(2+3='Bob')
```

The composite key's 13 bytes break down as `[tableID=1][type_tag=0x01][id=56]`. For an internal
node's cell, `[child page ID, 4 bytes]` follows instead of column data (a cell starting with a
composite key is common to both leaf and internal nodes).
