// src/components/Investigations/InvestigationDetailsModal.tsx
import React, { useState, useEffect } from 'react';
import { apiFetch } from '../../utils/apiFetch';
import './InvestigationDetailsModal.css';

interface Evidence {
  id: string;
  userNumber: number;
  description: string;
  imageData?: string;
  imageType?: string;
}

interface InvestigationRecord {
  id: string;
  dispute_id: string;
  title: string;
  description: string;
  createdAt: string;
  endsAt: string;
}

interface DisputeDetail {
  id: string;
  title: string;
  description: string;
  amount: number;
  opponent: string;
  cryptocurrency: string;
  createdAt: string;
  updatedAt: string;
  imageData?: string;
  imageType?: string;
  result: string;
  vote?: boolean;
}

interface Props {
  id: string;
  onClose: () => void;
  onCompleted: () => void;
}

export const InvestigationDetailsModal: React.FC<Props> = ({ id, onClose, onCompleted }) => {
  const [step, setStep] = useState<'details' | 'evidence' | 'vote'>('details');
  const [investigation, setInvestigation] = useState<InvestigationRecord | null>(null);
  const [dispute, setDispute] = useState<DisputeDetail | null>(null);
  const [evidences, setEvidences] = useState<Evidence[]>([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const [voteLoading, setVoteLoading] = useState(false);

  // Load investigation record and dispute details
  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const invRes = await apiFetch(`/api/v1/investigations/${id}`);
        if (!invRes.ok) throw new Error();
        const { data: invData } = await invRes.json() as { data: InvestigationRecord };
        setInvestigation(invData);
        const disRes = await apiFetch(`/api/v1/disputes/${invData.dispute_id}/evidence`);
        if (!disRes.ok) throw new Error();
        const { data: disData } = await disRes.json() as { data: DisputeDetail };
        setDispute(disData);
      } catch {
        // ignore
      }
      setLoading(false);
    })();
  }, [id]);

  // Load evidences
  useEffect(() => {
    if (step !== 'evidence' || !investigation) return;
    (async () => {
      setLoading(true);
      try {
        const res = await apiFetch(`/api/v1/evidence?dispute_id=${investigation.dispute_id}`);
        if (!res.ok) throw new Error();
        const { data } = await res.json() as { data: Evidence[] };
        setEvidences(data.map((ev, idx) => ({ ...ev, userNumber: idx + 1 })));
      } catch {
        setEvidences([]);
      }
      setLoading(false);
    })();
  }, [step, investigation]);

  const handleNext = () => {
    if (step === 'details') setStep('evidence');
    else if (step === 'evidence') {
      if (currentIndex < evidences.length - 1) setCurrentIndex(i => i + 1);
      else setStep('vote');
    }
  };

  const handleBack = () => {
    if (step === 'evidence' && currentIndex > 0) setCurrentIndex(i => i - 1);
    else if (step === 'vote') {
      setStep('evidence');
      setCurrentIndex(evidences.length - 1);
    }
  };

  const handleVote = async (choice: 'p1' | 'p2' | 'draw') => {
    if (!dispute) return;
    setVoteLoading(true);
    try {
      await apiFetch(`/api/v1/investigations/${investigation?.id}/vote?vote=${choice}`, {
        method: 'POST',
      });
      onCompleted();
      onClose();
    } catch {
      // ignore
    }
    setVoteLoading(false);
  };

  return (
    <div className="overlay">
      <div className="detail-card">
        <button className="close-btn" onClick={onClose}>×</button>
        {loading && <p>Загрузка…</p>}

        {!loading && dispute && step === 'details' && (
          <>
            <h3>{dispute.title}</h3>
            <p>{dispute.description}</p>
            <button className="btn-next" onClick={handleNext}>К доказательствам</button>
          </>
        )}

        {!loading && step === 'evidence' && evidences.length > 0 && (
          <>
            <h4>Доказательства пользователя {evidences[currentIndex].userNumber}</h4>
            <p>{evidences[currentIndex].description}</p>
            {evidences[currentIndex].imageData && (
              <img
                src={`data:${evidences[currentIndex].imageType};base64,${evidences[currentIndex].imageData}`}
                alt="Evidence"
                className="evidence-image"
              />
            )}
            <div className="nav-buttons">
              <button disabled={currentIndex === 0} onClick={handleBack}>Назад</button>
              <button onClick={handleNext}>Далее</button>
            </div>
          </>
        )}

        {!loading && step === 'vote' && (
          <>
            <div className="header-with-back">
              <button className="back-btn" onClick={handleBack}>←</button>
              <h4>Окончательное голосование</h4>
            </div>
            <div className="vote-buttons">
              <button onClick={() => handleVote('p1')} disabled={voteLoading}>Пользователь 1</button>
              <button onClick={() => handleVote('draw')} disabled={voteLoading}>Ничья</button>
              <button onClick={() => handleVote('p2')} disabled={voteLoading}>Пользователь 2</button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};