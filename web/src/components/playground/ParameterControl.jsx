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
import { Input, Slider, Switch, Tag, Typography } from '@douyinfe/semi-ui';
import { Ban, Hash, Repeat, Shuffle, Thermometer } from 'lucide-react';

const NumericParameter = ({
  icon,
  title,
  enabled,
  value,
  min,
  max,
  step,
  onToggle,
  onChange,
  disabled,
}) => (
  <div
    className={`rounded-lg bg-[var(--semi-color-bg-0)] p-3 shadow-[0_1px_3px_rgba(15,23,42,0.06)] transition-opacity ${
      !enabled || disabled ? 'opacity-55' : ''
    }`}
  >
    <div className='mb-2 flex items-center justify-between gap-2'>
      <div className='flex min-w-0 items-center gap-2'>
        <span className='flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-[var(--semi-color-fill-0)] text-[var(--semi-color-text-2)]'>
          {icon}
        </span>
        <div className='min-w-0'>
          <Typography.Text strong className='block truncate text-sm'>
            {title}
          </Typography.Text>
        </div>
      </div>
      <div className='flex shrink-0 items-center gap-2'>
        <Tag size='small' shape='circle'>
          {value}
        </Tag>
        <Switch
          checked={enabled}
          onChange={onToggle}
          size='small'
          disabled={disabled}
        />
      </div>
    </div>
    <Slider
      step={step}
      min={min}
      max={max}
      value={value}
      onChange={onChange}
      disabled={!enabled || disabled}
    />
  </div>
);

const InputParameter = ({
  icon,
  title,
  enabled,
  children,
  onToggle,
  disabled,
}) => (
  <div
    className={`rounded-lg bg-[var(--semi-color-bg-0)] p-3 shadow-[0_1px_3px_rgba(15,23,42,0.06)] transition-opacity ${
      !enabled || disabled ? 'opacity-55' : ''
    }`}
  >
    <div className='mb-2 flex items-center justify-between gap-2'>
      <div className='flex min-w-0 items-center gap-2'>
        <span className='flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-[var(--semi-color-fill-0)] text-[var(--semi-color-text-2)]'>
          {icon}
        </span>
        <div className='min-w-0'>
          <Typography.Text strong className='block truncate text-sm'>
            {title}
          </Typography.Text>
        </div>
      </div>
      <Switch
        checked={enabled}
        onChange={onToggle}
        size='small'
        disabled={disabled}
      />
    </div>
    {children}
  </div>
);

const ParameterControl = ({
  inputs,
  parameterEnabled,
  onInputChange,
  onParameterToggle,
  disabled = false,
}) => {
  return (
    <div className='space-y-3'>
      <NumericParameter
        icon={<Thermometer size={14} />}
        title='Temperature'
        enabled={parameterEnabled.temperature}
        value={inputs.temperature}
        min={0.1}
        max={1}
        step={0.1}
        onToggle={() => onParameterToggle('temperature')}
        onChange={(value) => onInputChange('temperature', value)}
        disabled={disabled}
      />

      <NumericParameter
        icon={<Repeat size={14} />}
        title='Frequency Penalty'
        enabled={parameterEnabled.frequency_penalty}
        value={inputs.frequency_penalty}
        min={-2}
        max={2}
        step={0.1}
        onToggle={() => onParameterToggle('frequency_penalty')}
        onChange={(value) => onInputChange('frequency_penalty', value)}
        disabled={disabled}
      />

      <NumericParameter
        icon={<Ban size={14} />}
        title='Presence Penalty'
        enabled={parameterEnabled.presence_penalty}
        value={inputs.presence_penalty}
        min={-2}
        max={2}
        step={0.1}
        onToggle={() => onParameterToggle('presence_penalty')}
        onChange={(value) => onInputChange('presence_penalty', value)}
        disabled={disabled}
      />

      <InputParameter
        icon={<Hash size={14} />}
        title='Max Tokens'
        enabled={parameterEnabled.max_tokens}
        onToggle={() => onParameterToggle('max_tokens')}
        disabled={disabled}
      >
        <Input
          placeholder='MaxTokens'
          name='max_tokens'
          required
          autoComplete='new-password'
          defaultValue={0}
          value={inputs.max_tokens}
          onChange={(value) => onInputChange('max_tokens', value)}
          className='!rounded-lg'
          disabled={!parameterEnabled.max_tokens || disabled}
        />
      </InputParameter>

      <InputParameter
        icon={<Shuffle size={14} />}
        title='Seed'
        enabled={parameterEnabled.seed}
        onToggle={() => onParameterToggle('seed')}
        disabled={disabled}
      >
        <Input
          placeholder='Seed'
          name='seed'
          autoComplete='new-password'
          value={inputs.seed || ''}
          onChange={(value) =>
            onInputChange('seed', value === '' ? null : value)
          }
          className='!rounded-lg'
          disabled={!parameterEnabled.seed || disabled}
        />
      </InputParameter>
    </div>
  );
};

export default ParameterControl;
