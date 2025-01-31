import { createDataFrame, FieldType } from '@grafana/data';

import { FlameGraphDataContainer, LevelItem } from './dataTransform';
import { walkTree } from './rendering';

function makeDataFrame(fields: Record<string, Array<number | string>>) {
  return createDataFrame({
    fields: Object.keys(fields).map((key) => ({
      name: key,
      values: fields[key],
      type: typeof fields[key][0] === 'string' ? FieldType.string : FieldType.number,
    })),
  });
}

type RenderData = {
  item: LevelItem;
  x: number;
  y: number;
  width: number;
  height: number;
  label: string;
  collapsed: boolean;
};

describe('walkTree', () => {
  it('correctly compute sizes for a single item', () => {
    const root: LevelItem = { start: 0, itemIndexes: [0], children: [], value: 100, level: 0 };
    const container = new FlameGraphDataContainer(makeDataFrame({ value: [100], level: [1], label: ['1'], self: [0] }));
    walkTree(root, 'children', container, 100, 0, 1, 100, (item, x, y, width, height, label, collapsed) => {
      expect(item).toEqual(root);
      expect(x).toEqual(0);
      expect(y).toEqual(0);
      expect(width).toEqual(99); // -1 for border
      expect(height).toEqual(22);
      expect(label).toEqual('1');
      expect(collapsed).toEqual(false);
    });
  });

  it('should render a multiple items', () => {
    const root: LevelItem = {
      start: 0,
      itemIndexes: [0],
      value: 100,
      level: 0,
      children: [
        { start: 0, itemIndexes: [1], children: [], value: 50, level: 1 },
        { start: 50, itemIndexes: [2], children: [], value: 50, level: 1 },
      ],
    };
    const container = new FlameGraphDataContainer(
      makeDataFrame({ value: [100, 50, 50], level: [0, 1, 1], label: ['1', '2', '3'], self: [0, 50, 50] })
    );
    const renderData: RenderData[] = [];
    walkTree(root, 'children', container, 100, 0, 1, 100, (item, x, y, width, height, label, collapsed) => {
      renderData.push({ item, x, y, width, height, label, collapsed });
    });
    expect(renderData).toEqual([
      { item: root, width: 99, height: 22, x: 0, y: 0, collapsed: false, label: '1' },
      { item: root.children[0], width: 49, height: 22, x: 0, y: 22, collapsed: false, label: '2' },
      { item: root.children[1], width: 49, height: 22, x: 50, y: 22, collapsed: false, label: '3' },
    ]);
  });

  it('should render a collapsed items', () => {
    const root: LevelItem = {
      start: 0,
      itemIndexes: [0],
      value: 100,
      level: 0,
      children: [
        { start: 0, itemIndexes: [1], children: [], value: 1, level: 1 },
        { start: 1, itemIndexes: [2], children: [], value: 1, level: 1 },
      ],
    };
    const container = new FlameGraphDataContainer(
      makeDataFrame({ value: [100, 1, 1], level: [0, 1, 1], label: ['1', '2', '3'], self: [0, 1, 1] })
    );
    const renderData: RenderData[] = [];
    walkTree(root, 'children', container, 100, 0, 1, 100, (item, x, y, width, height, label, collapsed) => {
      renderData.push({ item, x, y, width, height, label, collapsed });
    });
    expect(renderData).toEqual([
      { item: root, width: 99, height: 22, x: 0, y: 0, collapsed: false, label: '1' },
      { item: root.children[0], width: 1, height: 22, x: 0, y: 22, collapsed: true, label: '2' },
      { item: root.children[1], width: 1, height: 22, x: 1, y: 22, collapsed: true, label: '3' },
    ]);
  });

  it('skips too small items', () => {
    const root: LevelItem = {
      start: 0,
      itemIndexes: [0],
      value: 100,
      level: 0,
      children: [
        { start: 0, itemIndexes: [1], children: [], value: 0.1, level: 1 },
        { start: 1, itemIndexes: [2], children: [], value: 0.1, level: 1 },
      ],
    };
    const container = new FlameGraphDataContainer(
      makeDataFrame({ value: [100, 0.1, 0.1], level: [0, 1, 1], label: ['1', '2', '3'], self: [0, 0.1, 0.1] })
    );
    const renderData: RenderData[] = [];
    walkTree(root, 'children', container, 100, 0, 1, 100, (item, x, y, width, height, label, collapsed) => {
      renderData.push({ item, x, y, width, height, label, collapsed });
    });
    expect(renderData).toEqual([{ item: root, width: 99, height: 22, x: 0, y: 0, collapsed: false, label: '1' }]);
  });
});
