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
import { useTranslation } from 'react-i18next';
import LegalDocumentPage from '../../components/legal/LegalDocumentPage';
import privacyPolicyHtml from '../../content/legal/privacy-policy.html?raw';

const PrivacyPolicy = () => {
  const { t } = useTranslation();

  return (
    <LegalDocumentPage
      title={t('隐私政策')}
      apiEndpoint='/api/privacy-policy'
      styleId='token-factory-privacy-policy-styles'
      defaultContent={privacyPolicyHtml}
      defaultFormat='html'
    />
  );
};

export default PrivacyPolicy;
