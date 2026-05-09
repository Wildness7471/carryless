export function renderHeader(container, user, csrfToken) {
    const adminLink = user && user.is_admin
        ? '<a href="/admin">Admin</a>'
        : '';

    const authLinks = user
        ? `<a href="/inventory">Items</a>
           <a href="/categories">Categories</a>
           <a href="/packs">Packs</a>
           <a href="/trips">Trips</a>
           <a href="/account">Account</a>
           ${adminLink}
           <form class="logout-form" action="/logout" method="POST">
               <input type="hidden" name="csrf_token" value="${csrfToken || ''}">
               <button type="submit" class="btn btn-secondary">Logout</button>
           </form>`
        : `<a href="/login">Login</a>
           <a href="/register">Register</a>`;

    container.innerHTML = `
        <header class="header">
            <nav class="nav">
                <div class="nav-brand"><a href="/">Carryless</a></div>
                <div class="nav-links">${authLinks}</div>
            </nav>
        </header>`;
}
