import { RouterProvider } from 'react-router-dom';

import { AppProviders } from '@/app/providers/AppProviders';
import { appRouter } from '@/app/router';

/**
 * Корневой компонент web-клиента Grace.
 */
export function App() {
	return (
		<AppProviders>
			<RouterProvider router={appRouter} />
		</AppProviders>
	);
}
