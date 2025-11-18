// 完全参考 veadk-python 的配置
export default {
  extends: ['docus'],
  app: {
    baseURL: '/aster/'
  },
  image: {
    provider: 'none'
  },
  // 修复警告
  robots: {
    robotsTxt: false
  },
  llms: {
    domain: 'https://aster.local'
  }
}
