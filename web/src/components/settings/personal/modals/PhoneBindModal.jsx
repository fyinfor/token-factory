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
import { Button, Input, Modal } from '@douyinfe/semi-ui';
import { IconKey } from '@douyinfe/semi-icons';
import { Phone } from 'lucide-react';
import Turnstile from 'react-turnstile';

/**
 * 个人设置：绑定或修改手机号（短信验证码）。
 */
const PhoneBindModal = ({
  t,
  showPhoneBindModal,
  setShowPhoneBindModal,
  inputs,
  handleInputChange,
  sendPhoneVerificationCode,
  bindPhone,
  disablePhoneButton,
  phoneLoading,
  phoneCountdown,
  turnstileEnabled,
  turnstileSiteKey,
  setTurnstileToken,
}) => {
  return (
    <Modal
      title={
        <span className='flex items-center'>
          <Phone className='mr-2 text-blue-500' size={18} />
          {t('绑定手机')}
        </span>
      }
      visible={showPhoneBindModal}
      onCancel={() => setShowPhoneBindModal(false)}
      onOk={bindPhone}
      size={'small'}
      centered={true}
      maskClosable={false}
      className='modern-modal'
    >
      <span className='block space-y-4 py-4'>
        <span className='flex gap-3'>
          <Input
            placeholder={t('请输入手机号')}
            onChange={(value) => handleInputChange('phone', value)}
            name='phone'
            value={inputs.phone}
            size='large'
            className='!rounded-lg flex-1'
            prefix={<Phone size={16} />}
          />
          <Button
            onClick={sendPhoneVerificationCode}
            disabled={disablePhoneButton || phoneLoading}
            className='!rounded-lg'
            type='primary'
            theme='outline'
            size='large'
          >
            {disablePhoneButton
              ? `${t('重新发送')} (${phoneCountdown})`
              : t('获取验证码')}
          </Button>
        </span>

        <Input
          placeholder={t('输入手机号收到的验证码')}
          name='sms_verification_code'
          value={inputs.sms_verification_code}
          onChange={(value) =>
            handleInputChange('sms_verification_code', value)
          }
          size='large'
          className='!rounded-lg'
          prefix={<IconKey />}
        />

        {turnstileEnabled && (
          <span className='flex justify-center'>
            <Turnstile
              sitekey={turnstileSiteKey}
              onVerify={(token) => {
                setTurnstileToken(token);
              }}
            />
          </span>
        )}
      </span>
    </Modal>
  );
};

export default PhoneBindModal;
