/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

// ISO 3166-1 alpha-2 国家码 + 国际长途区号（E.164 dialing code）+ 旗帜 emoji + 中文名。
// 完整覆盖（240+ 国家/地区），供前端下拉选择使用。flag emoji 由国家码 2 字母转 Regional Indicator。

// 来自 ISO 3166-1 alpha-2；不存在的/有争议的少量地区仍保留以满足下拉完整性。
// 排序：按 region 分组（亚洲/欧洲/非洲/美洲/大洋洲/其他）+ 区号升序。

export const COUNTRY_DIAL_CODES = [
  // 中国（特殊放第一）
  { code: 'CN', name: '中国大陆', dial: '86', flag: '🇨🇳' },
  { code: 'HK', name: '中国香港', dial: '852', flag: '🇭🇰' },
  { code: 'MO', name: '中国澳门', dial: '853', flag: '🇲🇴' },
  { code: 'TW', name: '中国台湾', dial: '886', flag: '🇹🇼' },

  // 亚洲
  { code: 'AF', name: '阿富汗', dial: '93', flag: '🇦🇫' },
  { code: 'AM', name: '亚美尼亚', dial: '374', flag: '🇦🇲' },
  { code: 'AZ', name: '阿塞拜疆', dial: '994', flag: '🇦🇿' },
  { code: 'BD', name: '孟加拉国', dial: '880', flag: '🇧🇩' },
  { code: 'BH', name: '巴林', dial: '973', flag: '🇧🇭' },
  { code: 'BN', name: '文莱', dial: '673', flag: '🇧🇳' },
  { code: 'BT', name: '不丹', dial: '975', flag: '🇧🇹' },
  { code: 'GE', name: '格鲁吉亚', dial: '995', flag: '🇬🇪' },
  { code: 'ID', name: '印度尼西亚', dial: '62', flag: '🇮🇩' },
  { code: 'IL', name: '以色列', dial: '972', flag: '🇮🇱' },
  { code: 'IN', name: '印度', dial: '91', flag: '🇮🇳' },
  { code: 'IQ', name: '伊拉克', dial: '964', flag: '🇮🇶' },
  { code: 'IR', name: '伊朗', dial: '98', flag: '🇮🇷' },
  { code: 'JO', name: '约旦', dial: '962', flag: '🇯🇴' },
  { code: 'JP', name: '日本', dial: '81', flag: '🇯🇵' },
  { code: 'KG', name: '吉尔吉斯斯坦', dial: '996', flag: '🇰🇬' },
  { code: 'KH', name: '柬埔寨', dial: '855', flag: '🇰🇭' },
  { code: 'KP', name: '朝鲜', dial: '850', flag: '🇰🇵' },
  { code: 'KR', name: '韩国', dial: '82', flag: '🇰🇷' },
  { code: 'KW', name: '科威特', dial: '965', flag: '🇰🇼' },
  { code: 'KZ', name: '哈萨克斯坦', dial: '7', flag: '🇰🇿' },
  { code: 'LA', name: '老挝', dial: '856', flag: '🇱🇦' },
  { code: 'LB', name: '黎巴嫩', dial: '961', flag: '🇱🇧' },
  { code: 'LK', name: '斯里兰卡', dial: '94', flag: '🇱🇰' },
  { code: 'MM', name: '缅甸', dial: '95', flag: '🇲🇲' },
  { code: 'MN', name: '蒙古', dial: '976', flag: '🇲🇳' },
  { code: 'MV', name: '马尔代夫', dial: '960', flag: '🇲🇻' },
  { code: 'MY', name: '马来西亚', dial: '60', flag: '🇲🇾' },
  { code: 'NP', name: '尼泊尔', dial: '977', flag: '🇳🇵' },
  { code: 'OM', name: '阿曼', dial: '968', flag: '🇴🇲' },
  { code: 'PH', name: '菲律宾', dial: '63', flag: '🇵🇭' },
  { code: 'PK', name: '巴基斯坦', dial: '92', flag: '🇵🇰' },
  { code: 'PS', name: '巴勒斯坦', dial: '970', flag: '🇵🇸' },
  { code: 'QA', name: '卡塔尔', dial: '974', flag: '🇶🇦' },
  { code: 'RU', name: '俄罗斯', dial: '7', flag: '🇷🇺' },
  { code: 'SA', name: '沙特阿拉伯', dial: '966', flag: '🇸🇦' },
  { code: 'SG', name: '新加坡', dial: '65', flag: '🇸🇬' },
  { code: 'SY', name: '叙利亚', dial: '963', flag: '🇸🇾' },
  { code: 'TH', name: '泰国', dial: '66', flag: '🇹🇭' },
  { code: 'TJ', name: '塔吉克斯坦', dial: '992', flag: '🇹🇯' },
  { code: 'TL', name: '东帝汶', dial: '670', flag: '🇹🇱' },
  { code: 'TM', name: '土库曼斯坦', dial: '993', flag: '🇹🇲' },
  { code: 'TR', name: '土耳其', dial: '90', flag: '🇹🇷' },
  { code: 'UZ', name: '乌兹别克斯坦', dial: '998', flag: '🇺🇿' },
  { code: 'VN', name: '越南', dial: '84', flag: '🇻🇳' },
  { code: 'YE', name: '也门', dial: '967', flag: '🇾🇪' },

  // 欧洲
  { code: 'AD', name: '安道尔', dial: '376', flag: '🇦🇩' },
  { code: 'AL', name: '阿尔巴尼亚', dial: '355', flag: '🇦🇱' },
  { code: 'AT', name: '奥地利', dial: '43', flag: '🇦🇹' },
  { code: 'AX', name: '奥兰群岛', dial: '358', flag: '🇦🇽' },
  { code: 'BA', name: '波黑', dial: '387', flag: '🇧🇦' },
  { code: 'BE', name: '比利时', dial: '32', flag: '🇧🇪' },
  { code: 'BG', name: '保加利亚', dial: '359', flag: '🇧🇬' },
  { code: 'BY', name: '白俄罗斯', dial: '375', flag: '🇧🇾' },
  { code: 'CH', name: '瑞士', dial: '41', flag: '🇨🇭' },
  { code: 'CY', name: '塞浦路斯', dial: '357', flag: '🇨🇾' },
  { code: 'CZ', name: '捷克', dial: '420', flag: '🇨🇿' },
  { code: 'DE', name: '德国', dial: '49', flag: '🇩🇪' },
  { code: 'DK', name: '丹麦', dial: '45', flag: '🇩🇰' },
  { code: 'EE', name: '爱沙尼亚', dial: '372', flag: '🇪🇪' },
  { code: 'ES', name: '西班牙', dial: '34', flag: '🇪🇸' },
  { code: 'FI', name: '芬兰', dial: '358', flag: '🇫🇮' },
  { code: 'FO', name: '法罗群岛', dial: '298', flag: '🇫🇴' },
  { code: 'FR', name: '法国', dial: '33', flag: '🇫🇷' },
  { code: 'GB', name: '英国', dial: '44', flag: '🇬🇧' },
  { code: 'GI', name: '直布罗陀', dial: '350', flag: '🇬🇮' },
  { code: 'GR', name: '希腊', dial: '30', flag: '🇬🇷' },
  { code: 'HR', name: '克罗地亚', dial: '385', flag: '🇭🇷' },
  { code: 'HU', name: '匈牙利', dial: '36', flag: '🇭🇺' },
  { code: 'IE', name: '爱尔兰', dial: '353', flag: '🇮🇪' },
  { code: 'IM', name: '马恩岛', dial: '44', flag: '🇮🇲' },
  { code: 'IS', name: '冰岛', dial: '354', flag: '🇮🇸' },
  { code: 'IT', name: '意大利', dial: '39', flag: '🇮🇹' },
  { code: 'JE', name: '泽西岛', dial: '44', flag: '🇯🇪' },
  { code: 'LI', name: '列支敦士登', dial: '423', flag: '🇱🇮' },
  { code: 'LT', name: '立陶宛', dial: '370', flag: '🇱🇹' },
  { code: 'LU', name: '卢森堡', dial: '352', flag: '🇱🇺' },
  { code: 'LV', name: '拉脱维亚', dial: '371', flag: '🇱🇻' },
  { code: 'MC', name: '摩纳哥', dial: '377', flag: '🇲🇨' },
  { code: 'MD', name: '摩尔多瓦', dial: '373', flag: '🇲🇩' },
  { code: 'ME', name: '黑山', dial: '382', flag: '🇲🇪' },
  { code: 'MK', name: '北马其顿', dial: '389', flag: '🇲🇰' },
  { code: 'MT', name: '马耳他', dial: '356', flag: '🇲🇹' },
  { code: 'NL', name: '荷兰', dial: '31', flag: '🇳🇱' },
  { code: 'NO', name: '挪威', dial: '47', flag: '🇳🇴' },
  { code: 'PL', name: '波兰', dial: '48', flag: '🇵🇱' },
  { code: 'PT', name: '葡萄牙', dial: '351', flag: '🇵🇹' },
  { code: 'RO', name: '罗马尼亚', dial: '40', flag: '🇷🇴' },
  { code: 'RS', name: '塞尔维亚', dial: '381', flag: '🇷🇸' },
  { code: 'SE', name: '瑞典', dial: '46', flag: '🇸🇪' },
  { code: 'SI', name: '斯洛文尼亚', dial: '386', flag: '🇸🇮' },
  { code: 'SK', name: '斯洛伐克', dial: '421', flag: '🇸🇰' },
  { code: 'SM', name: '圣马力诺', dial: '378', flag: '🇸🇲' },
  { code: 'UA', name: '乌克兰', dial: '380', flag: '🇺🇦' },
  { code: 'VA', name: '梵蒂冈', dial: '379', flag: '🇻🇦' },
  { code: 'XK', name: '科索沃', dial: '383', flag: '🇽🇰' },

  // 非洲
  { code: 'AO', name: '安哥拉', dial: '244', flag: '🇦🇴' },
  { code: 'BF', name: '布基纳法索', dial: '226', flag: '🇧🇫' },
  { code: 'BI', name: '布隆迪', dial: '257', flag: '🇧🇮' },
  { code: 'BJ', name: '贝宁', dial: '229', flag: '🇧🇯' },
  { code: 'BW', name: '博茨瓦纳', dial: '267', flag: '🇧🇼' },
  { code: 'CD', name: '刚果（金）', dial: '243', flag: '🇨🇩' },
  { code: 'CF', name: '中非共和国', dial: '236', flag: '🇨🇫' },
  { code: 'CG', name: '刚果（布）', dial: '242', flag: '🇨🇬' },
  { code: 'CI', name: '科特迪瓦', dial: '225', flag: '🇨🇮' },
  { code: 'CM', name: '喀麦隆', dial: '237', flag: '🇨🇲' },
  { code: 'CV', name: '佛得角', dial: '238', flag: '🇨🇻' },
  { code: 'DJ', name: '吉布提', dial: '253', flag: '🇩🇯' },
  { code: 'DZ', name: '阿尔及利亚', dial: '213', flag: '🇩🇿' },
  { code: 'EG', name: '埃及', dial: '20', flag: '🇪🇬' },
  { code: 'EH', name: '西撒哈拉', dial: '212', flag: '🇪🇭' },
  { code: 'ER', name: '厄立特里亚', dial: '291', flag: '🇪🇷' },
  { code: 'ET', name: '埃塞俄比亚', dial: '251', flag: '🇪🇹' },
  { code: 'GA', name: '加蓬', dial: '241', flag: '🇬🇦' },
  { code: 'GH', name: '加纳', dial: '233', flag: '🇬🇭' },
  { code: 'GM', name: '冈比亚', dial: '220', flag: '🇬🇲' },
  { code: 'GN', name: '几内亚', dial: '224', flag: '🇬🇳' },
  { code: 'GQ', name: '赤道几内亚', dial: '240', flag: '🇬🇶' },
  { code: 'GW', name: '几内亚比绍', dial: '245', flag: '🇬🇼' },
  { code: 'KE', name: '肯尼亚', dial: '254', flag: '🇰🇪' },
  { code: 'KM', name: '科摩罗', dial: '269', flag: '🇰🇲' },
  { code: 'LR', name: '利比里亚', dial: '231', flag: '🇱🇷' },
  { code: 'LS', name: '莱索托', dial: '266', flag: '🇱🇸' },
  { code: 'LY', name: '利比亚', dial: '218', flag: '🇱🇾' },
  { code: 'MA', name: '摩洛哥', dial: '212', flag: '🇲🇦' },
  { code: 'MG', name: '马达加斯加', dial: '261', flag: '🇲🇬' },
  { code: 'ML', name: '马里', dial: '223', flag: '🇲🇱' },
  { code: 'MR', name: '毛里塔尼亚', dial: '222', flag: '🇲🇷' },
  { code: 'MU', name: '毛里求斯', dial: '230', flag: '🇲🇺' },
  { code: 'MW', name: '马拉维', dial: '265', flag: '🇲🇼' },
  { code: 'MZ', name: '莫桑比克', dial: '258', flag: '🇲🇿' },
  { code: 'NA', name: '纳米比亚', dial: '264', flag: '🇳🇦' },
  { code: 'NE', name: '尼日尔', dial: '227', flag: '🇳🇪' },
  { code: 'NG', name: '尼日利亚', dial: '234', flag: '🇳🇬' },
  { code: 'RW', name: '卢旺达', dial: '250', flag: '🇷🇼' },
  { code: 'SC', name: '塞舌尔', dial: '248', flag: '🇸🇨' },
  { code: 'SD', name: '苏丹', dial: '249', flag: '🇸🇩' },
  { code: 'SL', name: '塞拉利昂', dial: '232', flag: '🇸🇱' },
  { code: 'SN', name: '塞内加尔', dial: '221', flag: '🇸🇳' },
  { code: 'SO', name: '索马里', dial: '252', flag: '🇸🇴' },
  { code: 'SS', name: '南苏丹', dial: '211', flag: '🇸🇸' },
  { code: 'ST', name: '圣多美和普林西比', dial: '239', flag: '🇸🇹' },
  { code: 'SZ', name: '斯威士兰', dial: '268', flag: '🇸🇿' },
  { code: 'TD', name: '乍得', dial: '235', flag: '🇹🇩' },
  { code: 'TG', name: '多哥', dial: '228', flag: '🇹🇬' },
  { code: 'TN', name: '突尼斯', dial: '216', flag: '🇹🇳' },
  { code: 'TZ', name: '坦桑尼亚', dial: '255', flag: '🇹🇿' },
  { code: 'UG', name: '乌干达', dial: '256', flag: '🇺🇬' },
  { code: 'ZA', name: '南非', dial: '27', flag: '🇿🇦' },
  { code: 'ZM', name: '赞比亚', dial: '260', flag: '🇿🇲' },
  { code: 'ZW', name: '津巴布韦', dial: '263', flag: '🇿🇼' },

  // 美洲
  { code: 'AG', name: '安提瓜和巴布达', dial: '1', flag: '🇦🇬' },
  { code: 'AI', name: '安圭拉', dial: '1', flag: '🇦🇮' },
  { code: 'AR', name: '阿根廷', dial: '54', flag: '🇦🇷' },
  { code: 'AW', name: '阿鲁巴', dial: '297', flag: '🇦🇼' },
  { code: 'BB', name: '巴巴多斯', dial: '1', flag: '🇧🇧' },
  { code: 'BL', name: '圣巴泰勒米', dial: '590', flag: '🇧🇱' },
  { code: 'BM', name: '百慕大', dial: '1', flag: '🇧🇲' },
  { code: 'BO', name: '玻利维亚', dial: '591', flag: '🇧🇴' },
  { code: 'BQ', name: '荷属加勒比区', dial: '599', flag: '🇧🇶' },
  { code: 'BR', name: '巴西', dial: '55', flag: '🇧🇷' },
  { code: 'BS', name: '巴哈马', dial: '1', flag: '🇧🇸' },
  { code: 'BZ', name: '伯利兹', dial: '501', flag: '🇧🇿' },
  { code: 'CA', name: '加拿大', dial: '1', flag: '🇨🇦' },
  { code: 'CL', name: '智利', dial: '56', flag: '🇨🇱' },
  { code: 'CO', name: '哥伦比亚', dial: '57', flag: '🇨🇴' },
  { code: 'CR', name: '哥斯达黎加', dial: '506', flag: '🇨🇷' },
  { code: 'CU', name: '古巴', dial: '53', flag: '🇨🇺' },
  { code: 'CW', name: '库拉索', dial: '599', flag: '🇨🇼' },
  { code: 'DM', name: '多米尼克', dial: '1', flag: '🇩🇲' },
  { code: 'DO', name: '多米尼加共和国', dial: '1', flag: '🇩🇴' },
  { code: 'EC', name: '厄瓜多尔', dial: '593', flag: '🇪🇨' },
  { code: 'GD', name: '格林纳达', dial: '1', flag: '🇬🇩' },
  { code: 'GF', name: '法属圭亚那', dial: '594', flag: '🇬🇫' },
  { code: 'GL', name: '格陵兰', dial: '299', flag: '🇬🇱' },
  { code: 'GP', name: '瓜德罗普', dial: '590', flag: '🇬🇵' },
  { code: 'GT', name: '危地马拉', dial: '502', flag: '🇬🇹' },
  { code: 'GY', name: '圭亚那', dial: '592', flag: '🇬🇾' },
  { code: 'HN', name: '洪都拉斯', dial: '504', flag: '🇭🇳' },
  { code: 'HT', name: '海地', dial: '509', flag: '🇭🇹' },
  { code: 'JM', name: '牙买加', dial: '1', flag: '🇯🇲' },
  { code: 'KN', name: '圣基茨和尼维斯', dial: '1', flag: '🇰🇳' },
  { code: 'KY', name: '开曼群岛', dial: '1', flag: '🇰🇾' },
  { code: 'LC', name: '圣卢西亚', dial: '1', flag: '🇱🇨' },
  { code: 'MF', name: '法属圣马丁', dial: '590', flag: '🇲🇫' },
  { code: 'MQ', name: '马提尼克', dial: '596', flag: '🇲🇶' },
  { code: 'MS', name: '蒙特塞拉特', dial: '1', flag: '🇲🇸' },
  { code: 'MX', name: '墨西哥', dial: '52', flag: '🇲🇽' },
  { code: 'NI', name: '尼加拉瓜', dial: '505', flag: '🇳🇮' },
  { code: 'PA', name: '巴拿马', dial: '507', flag: '🇵🇦' },
  { code: 'PE', name: '秘鲁', dial: '51', flag: '🇵🇪' },
  { code: 'PM', name: '圣皮埃尔和密克隆', dial: '508', flag: '🇵🇲' },
  { code: 'PR', name: '波多黎各', dial: '1', flag: '🇵🇷' },
  { code: 'PY', name: '巴拉圭', dial: '595', flag: '🇵🇾' },
  { code: 'SR', name: '苏里南', dial: '597', flag: '🇸🇷' },
  { code: 'SV', name: '萨尔瓦多', dial: '503', flag: '🇸🇻' },
  { code: 'SX', name: '荷属圣马丁', dial: '1', flag: '🇸🇽' },
  { code: 'TC', name: '特克斯和凯科斯群岛', dial: '1', flag: '🇹🇨' },
  { code: 'TT', name: '特立尼达和多巴哥', dial: '1', flag: '🇹🇹' },
  { code: 'US', name: '美国', dial: '1', flag: '🇺🇸' },
  { code: 'UY', name: '乌拉圭', dial: '598', flag: '🇺🇾' },
  { code: 'VC', name: '圣文森特和格林纳丁斯', dial: '1', flag: '🇻🇨' },
  { code: 'VE', name: '委内瑞拉', dial: '58', flag: '🇻🇪' },
  { code: 'VG', name: '英属维尔京群岛', dial: '1', flag: '🇻🇬' },
  { code: 'VI', name: '美属维尔京群岛', dial: '1', flag: '🇻🇮' },

  // 大洋洲
  { code: 'AS', name: '美属萨摩亚', dial: '1', flag: '🇦🇸' },
  { code: 'AU', name: '澳大利亚', dial: '61', flag: '🇦🇺' },
  { code: 'CK', name: '库克群岛', dial: '682', flag: '🇨🇰' },
  { code: 'FJ', name: '斐济', dial: '679', flag: '🇫🇯' },
  { code: 'FM', name: '密克罗尼西亚', dial: '691', flag: '🇫🇲' },
  { code: 'GU', name: '关岛', dial: '1', flag: '🇬🇺' },
  { code: 'KI', name: '基里巴斯', dial: '686', flag: '🇰🇮' },
  { code: 'MH', name: '马绍尔群岛', dial: '692', flag: '🇲🇭' },
  { code: 'MP', name: '北马里亚纳群岛', dial: '1', flag: '🇲🇵' },
  { code: 'NC', name: '新喀里多尼亚', dial: '687', flag: '🇳🇨' },
  { code: 'NF', name: '诺福克岛', dial: '672', flag: '🇳🇫' },
  { code: 'NR', name: '瑙鲁', dial: '674', flag: '🇳🇷' },
  { code: 'NU', name: '纽埃', dial: '683', flag: '🇳🇺' },
  { code: 'NZ', name: '新西兰', dial: '64', flag: '🇳🇿' },
  { code: 'PF', name: '法属波利尼西亚', dial: '689', flag: '🇵🇫' },
  { code: 'PG', name: '巴布亚新几内亚', dial: '675', flag: '🇵🇬' },
  { code: 'PN', name: '皮特凯恩群岛', dial: '64', flag: '🇵🇳' },
  { code: 'PW', name: '帕劳', dial: '680', flag: '🇵🇼' },
  { code: 'SB', name: '所罗门群岛', dial: '677', flag: '🇸🇧' },
  { code: 'TK', name: '托克劳', dial: '690', flag: '🇹🇰' },
  { code: 'TO', name: '汤加', dial: '676', flag: '🇹🇴' },
  { code: 'TV', name: '图瓦卢', dial: '688', flag: '🇹🇻' },
  { code: 'VU', name: '瓦努阿图', dial: '678', flag: '🇻🇺' },
  { code: 'WF', name: '瓦利斯和富图纳', dial: '681', flag: '🇼🇫' },
  { code: 'WS', name: '萨摩亚', dial: '685', flag: '🇼🇸' },
];

// 按 ISO 国家码查找。
export const findCountryByCode = (code) =>
  COUNTRY_DIAL_CODES.find((c) => c.code === code);

// 拼接成 E.164：+{dial}{localNumber}
export const toE164 = (country, localNumber) => {
  if (!country) return '';
  const local = String(localNumber || '').replace(/[^\d]/g, '');
  return `+${country.dial}${local}`;
};

// 把 E.164 (+国码+号码) 拆成 { countryCode, localNumber }，匹配不到时回退 { CN, 原值 }。
// 主要用于：编辑表单回显「已存的手机号」时，把 E.164 字符串反拆成「国码 select + 本地号 input」。
export const splitE164Phone = (phone) => {
  const v = String(phone || '').trim();
  if (!v) return { countryCode: 'CN', localNumber: '' };
  if (v.startsWith('+')) {
    const rest = v.slice(1).replace(/[^\d]/g, '');
    // 优先匹配长前缀（3 位 → 2 位 → 1 位）
    for (const len of [3, 2, 1]) {
      const prefix = rest.slice(0, len);
      const hit = COUNTRY_DIAL_CODES.find((c) => c.dial === prefix);
      if (hit) {
        return {
          countryCode: hit.code,
          localNumber: rest.slice(len),
        };
      }
    }
  }
  // 国内 11 位默认归到 CN
  return { countryCode: 'CN', localNumber: v.replace(/[^\d]/g, '') };
};
