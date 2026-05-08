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

/**
 * 构建管理端「用户邮箱」字段校验规则：选填；填写时需为基本邮箱格式（服务端仍会做一次校验与占用检查）。
 *
 * @param {function} t i18n 翻译函数
 * @returns {Array<object>} Semi Form rules
 */
export function buildAdminUserEmailFieldRules(t) {
  return [
    {
      validator: (rule, value) => {
        const v = (value || '').trim();
        if (!v) return true;
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) {
          return new Error(t('请输入有效的邮箱地址'));
        }
        return true;
      },
    },
  ];
}
