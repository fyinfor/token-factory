/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import DistributorApplyFileUpload from './DistributorApplyFileUpload';

/** 提现资料上传：复用分销商申请页同款上传组件 */
export default function DistributorWithdrawDocUpload({
  label,
  labelExtra,
  required,
  url,
  onUrlChange,
  onPreview,
  compact = false,
  imagesOnly = false,
  allowPdf = false,
  hint,
}) {
  return (
    <DistributorApplyFileUpload
      label={label}
      labelExtra={labelExtra}
      required={required}
      url={url}
      onUrlChange={onUrlChange}
      maxCount={1}
      onPreview={onPreview}
      compact={compact}
      imagesOnly={imagesOnly}
      allowPdf={allowPdf}
      hint={hint}
    />
  );
}
