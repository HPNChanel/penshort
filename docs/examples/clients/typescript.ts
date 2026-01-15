/**
 * Penshort TypeScript Client (Minimal Example)
 * 
 * A zero-dependency fetch-based client for the Penshort API.
 * Copy this into your project and customize as needed.
 */

interface PenshortConfig {
    baseUrl: string;
    apiKey: string;
}

interface CreateLinkRequest {
    destination: string;
    alias?: string;
    redirect_type?: 301 | 302;
    expires_at?: string;
}

interface Link {
    id: string;
    short_code: string;
    short_url: string;
    destination: string;
    redirect_type: number;
    expires_at?: string;
    status: 'active' | 'expired' | 'disabled';
    click_count: number;
    created_at: string;
    updated_at: string;
}

interface LinkListResponse {
    data: Link[];
    pagination: {
        next_cursor?: string;
        has_more: boolean;
    };
}

interface AnalyticsResponse {
    link_id: string;
    period: { from: string; to: string };
    summary: { total_clicks: number; unique_visitors: number };
    breakdown: {
        daily?: Array<{ date: string; total_clicks: number; unique_visitors: number }>;
        referrers?: Array<{ domain: string; clicks: number }>;
        countries?: Array<{ code: string; name: string; clicks: number }>;
    };
}

class PenshortClient {
    private config: PenshortConfig;

    constructor(config: PenshortConfig) {
        this.config = config;
    }

    private async request<T>(
        method: string,
        path: string,
        body?: unknown
    ): Promise<T> {
        const response = await fetch(`${this.config.baseUrl}${path}`, {
            method,
            headers: {
                'Authorization': `Bearer ${this.config.apiKey}`,
                'Content-Type': 'application/json',
            },
            body: body ? JSON.stringify(body) : undefined,
        });

        // Handle rate limiting
        if (response.status === 429) {
            const retryAfter = response.headers.get('Retry-After');
            throw new Error(`Rate limited. Retry after ${retryAfter} seconds.`);
        }

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Request failed');
        }

        if (response.status === 204) {
            return {} as T;
        }

        return response.json();
    }

    // Links
    async createLink(data: CreateLinkRequest): Promise<Link> {
        return this.request<Link>('POST', '/api/v1/links', data);
    }

    async getLink(id: string): Promise<Link> {
        return this.request<Link>('GET', `/api/v1/links/${id}`);
    }

    async listLinks(params?: {
        cursor?: string;
        limit?: number;
        status?: string;
    }): Promise<LinkListResponse> {
        const query = new URLSearchParams();
        if (params?.cursor) query.set('cursor', params.cursor);
        if (params?.limit) query.set('limit', params.limit.toString());
        if (params?.status) query.set('status', params.status);

        const queryString = query.toString();
        return this.request<LinkListResponse>(
            'GET',
            `/api/v1/links${queryString ? `?${queryString}` : ''}`
        );
    }

    async updateLink(
        id: string,
        data: Partial<Pick<Link, 'destination' | 'redirect_type' | 'expires_at'> & { enabled?: boolean }>
    ): Promise<Link> {
        return this.request<Link>('PATCH', `/api/v1/links/${id}`, data);
    }

    async deleteLink(id: string): Promise<void> {
        await this.request<void>('DELETE', `/api/v1/links/${id}`);
    }

    // Analytics
    async getAnalytics(
        linkId: string,
        params?: { from?: string; to?: string; include?: string }
    ): Promise<AnalyticsResponse> {
        const query = new URLSearchParams();
        if (params?.from) query.set('from', params.from);
        if (params?.to) query.set('to', params.to);
        if (params?.include) query.set('include', params.include);

        const queryString = query.toString();
        return this.request<AnalyticsResponse>(
            'GET',
            `/api/v1/links/${linkId}/analytics${queryString ? `?${queryString}` : ''}`
        );
    }
}

// Usage Example
async function main() {
    const client = new PenshortClient({
        baseUrl: 'http://localhost:8080',
        apiKey: 'pk_live_your_key_here',
    });

    // Create a link
    const link = await client.createLink({
        destination: 'https://example.com',
        alias: 'my-link',
    });
    console.log('Created:', link.short_url);

    // Get analytics
    const analytics = await client.getAnalytics(link.id);
    console.log('Clicks:', analytics.summary.total_clicks);
}

export { PenshortClient, type PenshortConfig, type Link, type CreateLinkRequest };
