import React from 'react';
import RealNameVerificationCard from '../components/settings/personal/cards/RealNameVerificationCard';
import './RealNameVerification.css';

export default function RealNameVerification() {
  return (
    <div className='real-name-page'>
      <div className='real-name-page__ambient' />
      <div className='relative mx-auto w-full max-w-2xl'>
        <RealNameVerificationCard />
      </div>
    </div>
  );
}
