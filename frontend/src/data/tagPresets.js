// 预设的标签分组，用于筛选弹窗与壁纸标签编辑时的快捷选择。
// 标签本身作为数据存储（始终是中文原值），界面按当前语言显示对应译文。
export const TAG_GROUPS = [
  {
    key: 'style',
    labelKey: 'tagGroups.style',
    tags: ['写实', '动漫', '插画', '抽象', '极简', '赛博朋克', '像素', '水墨', '油画', '3D 渲染'],
  },
  {
    key: 'theme',
    labelKey: 'tagGroups.theme',
    tags: ['风景', '人物', '动物', '城市', '建筑', '太空', '自然', '汽车', '游戏', '美食'],
  },
  {
    key: 'tone',
    labelKey: 'tagGroups.tone',
    tags: ['深色', '浅色', '黑白', '冷色', '暖色', '高饱和', '低饱和'],
  },
  {
    key: 'usage',
    labelKey: 'tagGroups.usage',
    tags: ['桌面壁纸', '手机壁纸', '宽屏', '多屏拼接'],
  },
]

export const PRESET_TAGS = new Set(TAG_GROUPS.flatMap((group) => group.tags))
