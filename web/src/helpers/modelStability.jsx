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

import React from 'react';
import { Tag, Space, Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

/**
 * 与渠道列表「响应时间」标签相同的毫秒分档与颜色，用于模型广场/详情稳定性展示。
 * @param {number} responseTime 耗时（毫秒）
 * @param {function} t i18n
 * @returns {React.ReactNode}
 */
export const renderStabilityLatencyTag = (responseTime, t) => {
  if (responseTime === 0 || responseTime == null) {
    return (
      <Tag color='grey' shape='circle'>
        {t('未测')}
      </Tag>
    );
  }
  const sec = (responseTime / 1000).toFixed(2) + t(' 秒');
  if (responseTime <= 1000) {
    return (
      <Tag color='green' shape='circle'>
        {sec}
      </Tag>
    );
  }
  if (responseTime <= 3000) {
    return (
      <Tag color='lime' shape='circle'>
        {sec}
      </Tag>
    );
  }
  if (responseTime <= 5000) {
    return (
      <Tag color='yellow' shape='circle'>
        {sec}
      </Tag>
    );
  }
  return (
    <Tag color='red' shape='circle'>
      {sec}
    </Tag>
  );
};

const GRADE_TO_COLOR = {
  1: 'red',
  2: 'orange',
  3: 'yellow',
  4: 'lime',
  5: 'green',
};

/**
 * 稳定性等级文案：数据库 1-5（低到高）映射为 D/C/B/A/S，便于运营与用户理解。
 * @param {number} grade 1-5
 * @param {function} t i18n
 * @returns {string}
 */
export const getStabilityGradeLabel = (grade, t) => {
  const map = {
    1: t('D级（基础）'),
    2: t('C级（一般）'),
    3: t('B级（良好）'),
    4: t('A级（优秀）'),
    5: t('S级（卓越）'),
  };
  return map[grade] || t('未知等级');
};

/**
 * 无实测毫秒、仅有运营「稳定性等级」1-5 时的色块标签。
 * @param {number} grade 1-5
 * @param {function} t
 * @returns {React.ReactNode|null}
 */
export const renderStabilityGradeTag = (grade, t) => {
  if (!grade || grade < 1 || grade > 5) {
    return null;
  }
  const color = GRADE_TO_COLOR[grade] || 'grey';
  return (
    <Tag color={color} shape='circle' type='light'>
      {getStabilityGradeLabel(grade, t)}
    </Tag>
  );
};

/**
 * 将单条 DTO 渲染为「状态 + 稳定性耗时/等级」的紧凑行（供模型广场通道列表使用）。
 * @param {object} row 接口 /api/channel/model-test-results 的 data 元素
 * @param {function} t
 * @returns {React.ReactNode}
 */
export const renderModelTestResultSummary = (row, t) => {
  if (!row) {
    return (
      <Text type='tertiary' size='small'>
        {t('未测')}
      </Text>
    );
  }
  return (
    <Space wrap spacing={4} align='center'>
      {row.last_test_success ? (
        <Tag color='green' size='small' shape='circle' type='light'>
          {t('成功')}
        </Tag>
      ) : (
        <Tag color='red' size='small' shape='circle' type='light'>
          {t('失败')}
        </Tag>
      )}
      {row.display_response_time_ms > 0
        ? renderStabilityLatencyTag(row.display_response_time_ms, t)
        : null}
      {row.display_stability_grade > 0
        ? renderStabilityGradeTag(row.display_stability_grade, t)
        : null}
      {row.display_response_time_ms <= 0 && row.display_stability_grade <= 0 ? (
        <Text type='tertiary' size='small'>
          {t('无耗时')}
        </Text>
      ) : null}
    </Space>
  );
};
