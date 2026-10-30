// 气泡节点。课程页使用接口返回的树：章是子节点，叶子是 activity。

export interface BubbleNode {
  label: string
  note: string
  code?: string
  key?: string
  kind?: string
  activityType?: string
  questionId?: number
  children?: BubbleNode[]
}

export const sampleCourseTree: BubbleNode = {
  label: 'Data Structures and Database Systems',
  code: 'COMP5112',
  note: 'Three credits, level 5. Half of it is how values are arranged in memory; the other half is how a database is designed and reached.',
  children: [
    {
      label: 'Data structures',
      note: 'How values are arranged, and what that arrangement makes cheap.',
      children: [
        {
          label: 'Linear structures',
          note: 'Ordered by position — one neighbour before, one after.',
          children: [
            { label: 'Linked lists', note: 'Nodes chained by pointers, so an insert costs O(1) once the position is known.' },
            { label: 'Stacks', note: 'Last in, first out: push, pop and peek, all at one end.' },
            { label: 'Queues', note: 'First in, first out: enqueue at the rear, dequeue at the front.' },
          ],
        },
        {
          label: 'Tree structures',
          note: 'Hierarchical — each node has one parent and ordered children.',
          children: [
            { label: 'Binary trees', note: 'At most two children; search, insert and delete all cost O(h).' },
            { label: 'Balanced trees', note: 'Rotations on insert and delete keep the height at O(log n).' },
          ],
        },
        {
          label: 'Priority queues',
          note: 'The highest-priority item leaves first, not the oldest.',
          children: [
            { label: 'Binary heap', note: 'A complete tree stored in an array, with every parent below its children.' },
            { label: 'Heap operations', note: 'Insert and extract-min in O(log n); building a heap from scratch in O(n).' },
          ],
        },
      ],
    },
    {
      label: 'Sorting and searching',
      note: 'Putting a collection in order, and finding one value inside it.',
      children: [
        {
          label: 'Elementary sorts',
          note: 'Quadratic, simple, and still the fastest thing on small input.',
          children: [
            { label: 'Bubble & insertion', note: 'Adjacent swaps and shifting: O(n²), but insertion sort wins on nearly sorted data.' },
            { label: 'Selection sort', note: 'Take the minimum each pass and swap once — O(n²) with the fewest writes.' },
          ],
        },
        {
          label: 'Divide and conquer',
          note: 'Split the range, sort each half, put them back together.',
          children: [
            { label: 'Quick sort', note: 'Partition around a pivot: O(n log n) on average, O(n²) if the pivot is always the worst.' },
            { label: 'Merge sort', note: 'Stable and always O(n log n), at the price of O(n) extra space.' },
          ],
        },
        {
          label: 'Searching',
          note: 'Finding one key among many.',
          children: [
            { label: 'Linear search', note: 'Look at everything — O(n), and the only option on unsorted data.' },
            { label: 'Binary search', note: 'Halve a sorted range each step, so O(log n) comparisons.' },
          ],
        },
      ],
    },
    {
      label: 'Database foundations',
      note: 'Data shared between applications, described once and queried by value.',
      children: [
        {
          label: 'The DBMS',
          note: 'A layer that manages what a pile of files cannot.',
          children: [
            { label: 'Files vs DBMS', note: 'Redundancy, inconsistency and concurrent access are the problems a DBMS exists to solve.' },
            { label: 'Components', note: 'A query processor above, a storage manager below, and one schema shared between them.' },
            { label: 'Independence', note: 'Logical and physical schemas can change without every program being rewritten.' },
          ],
        },
        {
          label: 'Relational model',
          note: 'Data as relations, reached by value rather than by pointer.',
          children: [
            { label: 'Relations & keys', note: 'Tuples and attributes, with candidate and primary keys naming each row.' },
            { label: 'Relational algebra', note: 'Selection, projection, join and the set operations — the semantics under SQL.' },
          ],
        },
        {
          label: 'SQL',
          note: 'The declarative language over the relational model.',
          children: [
            { label: 'DDL & DML', note: 'CREATE and ALTER define the shape; SELECT, INSERT and DELETE act on the rows.' },
            { label: 'Queries & joins', note: 'Joins combine relations; grouping and aggregation summarise them.' },
          ],
        },
      ],
    },
    {
      label: 'Design and storage',
      note: 'From a written requirement to tables that hold up under load.',
      children: [
        {
          label: 'ER modelling',
          note: 'Entities, their attributes, and the relationships between them.',
          children: [
            { label: 'Entities', note: 'Rectangles, diamonds and participation — the sketch before any table exists.' },
            { label: 'Cardinality', note: 'One-to-one, one-to-many, many-to-many, and weak entities that borrow a key.' },
          ],
        },
        {
          label: 'Normalization',
          note: 'Removing redundancy by decomposing, without losing information.',
          children: [
            { label: 'Dependencies', note: 'X determines Y; take the closure, then derive the candidate keys.' },
            { label: '3NF & BCNF', note: 'Every determinant is a key; the decomposition must be lossless.' },
          ],
        },
        {
          label: 'Indexing',
          note: 'Where a row physically sits, and which structure finds it.',
          children: [
            { label: 'File organization', note: 'Heap, sequential or clustered — the cost of every scan follows from this.' },
            { label: 'B+ trees & hashing', note: 'A tree for range queries, a hash for equality; both trade space for I/O.' },
          ],
        },
        {
          label: 'Query processing',
          note: 'From SQL text to a plan the storage manager can execute.',
          children: [
            { label: 'Query plans', note: 'Parse, rewrite, then choose an access path per relation.' },
            { label: 'Cost estimation', note: 'Estimate I/O and compare plans — join order usually decides the outcome.' },
          ],
        },
      ],
    },
  ],
}

export const bubbleLevelName = ['Course', 'Module', 'Topic', 'Concept']
