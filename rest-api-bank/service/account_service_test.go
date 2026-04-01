package service_test

// import (
// 	"testing"

// 	"rest-api-bank/mocks"
// 	"rest-api-bank/models"
// 	"rest-api-bank/service"

// 	"github.com/google/uuid"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func TestAccountService_Create(t *testing.T) {

// 	type Mocker struct {
// 		repo *mocks.AccountRepository
// 	}

// 	testCases := []struct {
// 		desc      string
// 		input     models.Account
// 		mockSetup func(m *Mocker)
// 		wantErr   bool
// 	}{
// 		{
// 			desc: "SUCCESS",
// 			input: models.Account{
// 				AccountNumber: "123",
// 				AccountHolder: "Fadhil",
// 				Balance:       100,
// 			},
// 			wantErr: false,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("Create", mock.Anything).Return(nil)
// 			},
// 		},
// 		{
// 			desc: "ERROR: empty account number",
// 			input: models.Account{
// 				AccountHolder: "Fadhil",
// 			},
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 		{
// 			desc: "ERROR: empty account holder",
// 			input: models.Account{
// 				AccountNumber: "123",
// 			},
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 		{
// 			desc: "ERROR: negative balance",
// 			input: models.Account{
// 				AccountNumber: "123",
// 				AccountHolder: "Fadhil",
// 				Balance:       -10,
// 			},
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 		{
// 			desc: "ERROR: repo fail",
// 			input: models.Account{
// 				AccountNumber: "123",
// 				AccountHolder: "Fadhil",
// 				Balance:       100,
// 			},
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("Create", mock.Anything).Return(assert.AnError)
// 			},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {
// 			m := &Mocker{
// 				repo: mocks.NewAccountRepository(t),
// 			}

// 			tC.mockSetup(m)

// 			svc := service.AccountService{Repo: m.repo}

// 			err := svc.Create(tC.input)

// 			if tC.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}

// 			m.repo.AssertExpectations(t)
// 		})
// 	}
// }

// func TestAccountService_GetAll(t *testing.T) {

// 	type Mocker struct {
// 		repo *mocks.AccountRepository
// 	}

// 	testCases := []struct {
// 		desc      string
// 		mockSetup func(m *Mocker)
// 		wantErr   bool
// 	}{
// 		{
// 			desc:    "SUCCESS",
// 			wantErr: false,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("GetAll").Return([]models.Account{}, nil)
// 			},
// 		},
// 		{
// 			desc:    "ERROR",
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("GetAll").Return(nil, assert.AnError)
// 			},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {

// 			m := &Mocker{repo: mocks.NewAccountRepository(t)}
// 			tC.mockSetup(m)

// 			svc := service.AccountService{Repo: m.repo}

// 			_, err := svc.GetAll()

// 			if tC.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}

// 			m.repo.AssertExpectations(t)
// 		})
// 	}
// }

// func TestAccountService_GetByID(t *testing.T) {

// 	type Mocker struct {
// 		repo *mocks.AccountRepository
// 	}

// 	testCases := []struct {
// 		desc      string
// 		id        string
// 		mockSetup func(m *Mocker)
// 		wantErr   bool
// 	}{
// 		{
// 			desc:    "SUCCESS",
// 			id:      uuid.New().String(),
// 			wantErr: false,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("GetByID", mock.Anything).
// 					Return(models.Account{}, nil)
// 			},
// 		},
// 		{
// 			desc:    "ERROR: empty id",
// 			id:      "",
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {

// 			m := &Mocker{repo: mocks.NewAccountRepository(t)}
// 			tC.mockSetup(m)

// 			svc := service.AccountService{Repo: m.repo}

// 			_, err := svc.GetByID(tC.id)

// 			if tC.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}

// 			m.repo.AssertExpectations(t)
// 		})
// 	}
// }

// func TestAccountService_Update(t *testing.T) {

// 	type Mocker struct {
// 		repo *mocks.AccountRepository
// 	}

// 	testCases := []struct {
// 		desc      string
// 		input     models.Account
// 		mockSetup func(m *Mocker)
// 		wantErr   bool
// 	}{
// 		{
// 			desc: "SUCCESS",
// 			input: models.Account{
// 				ID:            uuid.New(),
// 				AccountHolder: "Fadhil",
// 				Balance:       100,
// 			},
// 			wantErr: false,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("Update", mock.Anything).Return(nil)
// 			},
// 		},
// 		{
// 			desc: "ERROR: empty id",
// 			input: models.Account{},
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {

// 			m := &Mocker{repo: mocks.NewAccountRepository(t)}
// 			tC.mockSetup(m)

// 			svc := service.AccountService{Repo: m.repo}

// 			err := svc.Update(tC.input)

// 			if tC.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}

// 			m.repo.AssertExpectations(t)
// 		})
// 	}
// }

// func TestAccountService_Delete(t *testing.T) {

// 	type Mocker struct {
// 		repo *mocks.AccountRepository
// 	}

// 	testCases := []struct {
// 		desc      string
// 		id        string
// 		mockSetup func(m *Mocker)
// 		wantErr   bool
// 	}{
// 		{
// 			desc:    "SUCCESS",
// 			id:      uuid.New().String(),
// 			wantErr: false,
// 			mockSetup: func(m *Mocker) {
// 				m.repo.On("Delete", mock.Anything).Return(nil)
// 			},
// 		},
// 		{
// 			desc:    "ERROR: empty id",
// 			id:      "",
// 			wantErr: true,
// 			mockSetup: func(m *Mocker) {},
// 		},
// 	}

// 	for _, tC := range testCases {
// 		t.Run(tC.desc, func(t *testing.T) {

// 			m := &Mocker{repo: mocks.NewAccountRepository(t)}
// 			tC.mockSetup(m)

// 			svc := service.AccountService{Repo: m.repo}

// 			err := svc.Delete(tC.id)

// 			if tC.wantErr {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}

// 			m.repo.AssertExpectations(t)
// 		})
// 	}
// }