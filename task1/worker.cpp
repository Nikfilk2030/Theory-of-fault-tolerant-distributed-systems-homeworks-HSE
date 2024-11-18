#include "utils.h"

double function(const double x) {
    return x * x * x;
}

double count_integral(const double x, const double step) {
    return function(x) * step;
}

double get_integral(const TTask& task) {
    log_worker("starting integral calculation: " + task.as_string());

    double result = 0.0;
    for (double x = task.get_start(); x < task.get_end(); x += task.get_step()) {
        result += count_integral(x, task.get_step());
    }
    return result;
}

struct TInputParams {
    int discover_port = 0;
    int task_port = 0;

    TInputParams(int argc, char* argv[]) {
        try {
            if (argc != 3) {
                throw std::runtime_error("ERROR Wrong number of arguments");
            }
            discover_port = std::stoi(argv[1]);
            task_port = std::stoi(argv[2]);
        } catch (const std::exception& e) {
            std::cerr << e.what() << std::endl;
            exit(1);
        }
    }

    TString as_string() const {
        return "discover_port: " + std::to_string(discover_port) + ", task_port: " + std::to_string(task_port);
    }
};

int main(int argc, char* argv[]) {
    TInputParams params(argc, argv);

    log_worker("starting " + params.as_string());

    runWorker(params.discover_port, params.task_port);
}
