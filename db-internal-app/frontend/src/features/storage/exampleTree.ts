import type { TreeSnapshot } from '../../shared/types'

/** 実データに繋がっていない、説明用の「イメージ図」。実際のB+Treeは1ページに
 * 数十〜百件超入るため、相当な件数を入れない限りRoot+葉2枚という平べったい
 * 形にしかならない(TreeDiagramの実データ描画で確認済み)。3階層に育った
 * B+Treeがどんな見た目になるかを、TreeDiagramと同じ描画コード(layoutTree)に
 * 架空のTreeSnapshotを渡すことで示す。実データとは完全に独立している。 */
export const EXAMPLE_TREE: TreeSnapshot = {
  rootPageId: 1,
  pages: {
    1: { pageId: 1, isLeaf: false, keys: ['500'], childPageIds: [2], rightmostChild: 3 },
    2: { pageId: 2, isLeaf: false, keys: ['200'], childPageIds: [4], rightmostChild: 5 },
    3: { pageId: 3, isLeaf: false, keys: ['800'], childPageIds: [6], rightmostChild: 7 },
    4: {
      pageId: 4,
      isLeaf: true,
      rows: [
        [10, 'Alice'],
        [50, 'Bob'],
        [120, 'Carol'],
      ],
      nextLeafId: 5,
    },
    5: {
      pageId: 5,
      isLeaf: true,
      rows: [
        [220, 'Dave'],
        [310, 'Eve'],
        [480, 'Frank'],
      ],
      nextLeafId: 6,
    },
    6: {
      pageId: 6,
      isLeaf: true,
      rows: [
        [510, 'Grace'],
        [650, 'Heidi'],
        [790, 'Ivan'],
      ],
      nextLeafId: 7,
    },
    7: {
      pageId: 7,
      isLeaf: true,
      rows: [
        [820, 'Judy'],
        [900, 'Mallory'],
        [999, 'Niaj'],
      ],
    },
  },
}
