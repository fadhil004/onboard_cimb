package service_test

import (
	"testing"

	"rest-api-bank/mocks"
	"rest-api-bank/models"
	"rest-api-bank/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransferService_Transfer(t *testing.T) {

	id := uuid.New()
	type Mocker struct {
		accRepo *mocks.AccountRepository
		txRepo  *mocks.TransactionRepository
	}

	testCases := []struct {
		desc      string
		fromID    uuid.UUID
		toID      uuid.UUID
		amount    int64
		mockSetup func(m *Mocker)
		wantErr   bool
	}{
		{
			desc:    "SUCCESS: transfer success",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  100,
			wantErr: false,
			mockSetup: func(m *Mocker) {
				from := models.Account{Balance: 200}
				to := models.Account{Balance: 100}

				m.accRepo.On("GetByID", mock.Anything).Return(from, nil).Once()
				m.accRepo.On("GetByID", mock.Anything).Return(to, nil).Once()

				m.accRepo.On("Update", mock.Anything).Return(nil)
				m.txRepo.On("Create", mock.Anything).Return(nil)
			},
		},
		
		{
			desc:    "ERROR: same account",
			fromID:  id,
			toID:    id,
			amount:  100,
			wantErr: true,
			mockSetup: func(m *Mocker) {},
		},
		{
			desc:    "ERROR: invalid amount",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  0,
			wantErr: true,
			mockSetup: func(m *Mocker) {},
		},
		{
			desc:    "ERROR: insufficient balance",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  500,
			wantErr: true,
			mockSetup: func(m *Mocker) {
				m.accRepo.On("GetByID", mock.Anything).
					Return(models.Account{Balance: 100}, nil).Once()
				m.accRepo.On("GetByID", mock.Anything).
					Return(models.Account{Balance: 100}, nil).Once()
			},
		},
		{
			desc:    "ERROR: update from failed",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  100,
			wantErr: true,
			mockSetup: func(m *Mocker) {
				from := models.Account{Balance: 200}
				to := models.Account{Balance: 100}

				m.accRepo.On("GetByID", mock.Anything).Return(from, nil).Once()
				m.accRepo.On("GetByID", mock.Anything).Return(to, nil).Once()

				m.accRepo.On("Update", mock.Anything).Return(assert.AnError)
			},
		},
		{
			desc:    "ERROR: update to failed (rollback)",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  100,
			wantErr: true,
			mockSetup: func(m *Mocker) {
				from := models.Account{Balance: 200}
				to := models.Account{Balance: 100}

				m.accRepo.On("GetByID", mock.Anything).Return(from, nil).Once()
				m.accRepo.On("GetByID", mock.Anything).Return(to, nil).Once()

				m.accRepo.On("Update", mock.Anything).
					Return(nil).Once() // from

				m.accRepo.On("Update", mock.Anything).
					Return(assert.AnError).Once() // to

				m.accRepo.On("Update", mock.Anything).
					Return(nil).Once() // rollback
			},
		},
		{
			desc:    "ERROR: create transaction failed (rollback)",
			fromID:  uuid.New(),
			toID:    uuid.New(),
			amount:  100,
			wantErr: true,
			mockSetup: func(m *Mocker) {
				from := models.Account{Balance: 200}
				to := models.Account{Balance: 100}

				m.accRepo.On("GetByID", mock.Anything).Return(from, nil).Once()
				m.accRepo.On("GetByID", mock.Anything).Return(to, nil).Once()

				m.accRepo.On("Update", mock.Anything).Return(nil)
				m.txRepo.On("Create", mock.Anything).Return(assert.AnError)
			},
		},
	}

	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {

			m := &Mocker{
				accRepo: mocks.NewAccountRepository(t),
				txRepo:  mocks.NewTransactionRepository(t),
			}

			tC.mockSetup(m)

			svc := service.TransferService{
				AccountRepo:     m.accRepo,
				TransactionRepo: m.txRepo,
			}

			err := svc.Transfer(tC.fromID, tC.toID, tC.amount)

			if tC.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			m.accRepo.AssertExpectations(t)
			m.txRepo.AssertExpectations(t)
		})
	}
}