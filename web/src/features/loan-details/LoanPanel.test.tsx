import {render,screen,waitFor} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {QueryClient,QueryClientProvider} from '@tanstack/react-query'
import {beforeEach,expect,it,vi} from 'vitest'
import {LoanPanel} from './LoanPanel'

const {get,post,patch}=vi.hoisted(()=>({get:vi.fn(),post:vi.fn(),patch:vi.fn()}))
vi.mock('@/shared/api/client',()=>({apiClient:{GET:get,POST:post,PATCH:patch}}))
const draft={id:'copy',revision:1,appliedAt:null,baseHash:'snapshot',payments:[{sourceId:'paid',dueOn:'2026-10-01',amountMinor:10000,paidMinor:2500},{sourceId:'future',dueOn:'2026-11-01',amountMinor:10000,paidMinor:0}]}
beforeEach(()=>{
 get.mockResolvedValue({data:{lender:'',agreementNumber:'',agreementOn:null,principalMinor:null,annualRateBasisPoints:null,reminders:false,revision:0,documents:[],drafts:[]}})
 post.mockReset();post.mockResolvedValue({data:draft});patch.mockReset()
})
it('locks paid rows and requires saved edits plus separate confirmation before applying',async()=>{
 render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><LoanPanel obligationId="loan"/></QueryClientProvider>)
 await userEvent.click(screen.getByRole('button',{name:'Договор, документы и копии графика'}))
 await userEvent.click(await screen.findByRole('button',{name:'Скопировать текущий график'}))
 expect(await screen.findByLabelText('Дата 1 · есть оплата')).toBeDisabled()
 expect(screen.getAllByLabelText('Сумма')[0]).toBeDisabled()
 expect(screen.getByRole('button',{name:'Удалить платёж 1'})).toBeDisabled()
 await userEvent.clear(screen.getAllByLabelText('Сумма')[1])
 await userEvent.type(screen.getAllByLabelText('Сумма')[1],'125')
 expect(screen.getByRole('button',{name:'Применить сохранённый график'})).toBeDisabled()
 patch.mockResolvedValue({data:{...draft,revision:2,payments:[draft.payments[0],{...draft.payments[1],amountMinor:12500}]}})
 await userEvent.click(screen.getByRole('button',{name:'Сохранить черновик'}))
 await waitFor(()=>expect(screen.getByRole('button',{name:'Применить сохранённый график'})).toBeEnabled())
 await userEvent.click(screen.getByRole('button',{name:'Применить сохранённый график'}))
 expect(post).toHaveBeenCalledTimes(1)
 await userEvent.click(screen.getByRole('button',{name:'Подтвердить замену'}))
 await waitFor(()=>expect(post).toHaveBeenCalledWith('/api/v1/loans/{obligationId}/schedule-drafts/{draftId}/apply',expect.objectContaining({body:{revision:2,confirm:true}})))
})
