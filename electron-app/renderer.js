const goalForm = document.getElementById('goalForm');
const goalInput = document.getElementById('goalInput');
const outputDiv = document.getElementById('output');

goalForm.addEventListener('submit', (e) => {
    e.preventDefault();
    const goal = goalInput.value.trim();
    if (goal === '') {
        alert('Please enter a goal.');
        return;
    }

    fetch('http://localhost:8080/execute', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ goal })
    })
        .then(response => response.json())
        .then(data => {
            outputDiv.textContent = data.message;
        })
        .catch(error => {
            console.error('Error:', error);
            outputDiv.textContent = 'An error occurred. Check the console for details.';
        });
});
