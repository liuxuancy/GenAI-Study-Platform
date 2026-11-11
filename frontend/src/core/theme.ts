// Ant Design 主题：跟登录页的设计保持一致，黑白、直角、衬线字体。
import type { ThemeConfig } from 'antd'

export const appTheme: ThemeConfig = {
  token: {
    colorPrimary: '#111114',
    colorLink: '#111114',
    colorLinkHover: '#c0442c',
    colorError: '#c0442c',
    colorBorder: '#e8e8ec',
    colorBorderSecondary: '#e8e8ec',
    colorTextSecondary: '#55555e',
    borderRadius: 0,
    fontFamily: "'Source Serif 4', Georgia, 'Times New Roman', serif",
    fontSize: 15,
  },
  components: {
    Layout: { headerBg: '#ffffff', siderBg: '#ffffff', bodyBg: '#fafafb' },
    Menu: { itemSelectedBg: '#111114', itemSelectedColor: '#ffffff', itemBorderRadius: 0 },
    Select: {
      borderRadius: 9,
      colorBorder: '#e4e9e5',
      optionActiveBg: '#f7f8f5',
      optionSelectedBg: '#f8e1d5',
      optionFontSize: 12,
    },
  },
}
